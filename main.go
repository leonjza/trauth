package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	htpasswd "github.com/tg123/go-htpasswd"
)

var (
	serverPort           string
	domain               string
	cookiePath           string
	cookieName           string
	cookieSecure         bool
	cookieHttpOnly       bool
	passwordFileLocation string
	sessionKey           string
	realm                string

	passwordFile *htpasswd.File
	store        *sessions.CookieStore
)

// User holds a users account information
type User struct {
	Username      string
	Authenticated bool
}

func getUser(s *sessions.Session) User {
	val := s.Values["user"]
	var user = User{}
	user, ok := val.(User)
	if !ok {
		return User{Authenticated: false}
	}
	return user
}

func check(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, cookieName)
		if err != nil {
			log.Printf("unable to get session with error: %s", err)
		}
		user := getUser(session)

		// prefer the x-forwarded-for header for the remote address
		remoteAddr := r.Header.Get(`X-Forwarded-For`)
		if remoteAddr == `` {
			remoteAddr = r.RemoteAddr
		}

		if auth := user.Authenticated; !auth {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
			user, pass, ok := r.BasicAuth()

			if !ok {
				log.Printf("no basic auth creds provided from %s\n", remoteAddr)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			if !passwordFile.Match(user, pass) {
				log.Printf("invalid basic auth creds provided from %s\n", remoteAddr)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			session.Values["user"] = &User{
				Username:      user,
				Authenticated: true,
			}
			if err := session.Save(r, w); err != nil {
				log.Fatalf("failed to save session data with: %s\n", err)
			}

			newDestination := fmt.Sprintf("%s://%s:%s%s",
				r.Header.Get("X-Forwarded-Proto"),
				r.Header.Get("X-Forwarded-Host"),
				r.Header.Get("X-Forwarded-Port"),
				r.Header.Get("X-Forwarded-URI"))
			log.Printf("authenticated %s from %s using basic auth, redirecting to %s",
				user, remoteAddr, newDestination)

			http.Redirect(w, r, newDestination, http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Ok returns an OK
func Ok(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, cookieName)
	if err != nil {
		log.Printf("failed to read session cookie info with: %s\n", err)
		http.Error(w, `Internal Error`, http.StatusInternalServerError)
		return
	}

	user := getUser(session)
	fmt.Fprintf(w, "hello, %s. you should now be authenticated for %s!", user.Username, domain)
}

func parseEnv() error {
	serverPort = os.Getenv("TRAUTH_SERVER_PORT")
	domain = os.Getenv("TRAUTH_DOMAIN")
	realm = os.Getenv("TRAUTH_REALM")
	sessionKey = os.Getenv("TRAUTH_SESSION_KEY")
	passwordFileLocation = os.Getenv("TRAUTH_PASSWORD_FILE_LOCATION")

	// cookie setup
	cookiePath = os.Getenv("TRAUTH_COOKIE_PATH")
	cookieName = os.Getenv("TRAUTH_COOKIE_NAME")
	cookieSecureEnv, err := strconv.ParseBool(os.Getenv("TRAUTH_COOKIE_SECURE"))
	if err != nil {
		log.Printf(`warn: TRAUTH_COOKIE_SECURE is empty or is invalid, defaulting to false`)
		cookieSecure = false // default
	} else {
		cookieSecure = cookieSecureEnv
	}
	cookieHttpOnlyEnv, err := strconv.ParseBool(os.Getenv("TRAUTH_COOKIE_HTTPONLY"))
	if err != nil {
		log.Printf(`warn: TRAUTH_COOKIE_HTTPONLY is empty or is invalid, defaulting to false`)
		cookieHttpOnly = false // default
	} else {
		cookieHttpOnly = cookieHttpOnlyEnv
	}

	// value parsing, setting defaults if needed
	if serverPort == "" {
		serverPort = "8080"
	}

	if domain == "" {
		return errors.New("empty domain")
	}

	if cookiePath == "" {
		cookiePath = "/"
	}

	if cookieName == "" {
		cookieName = "trauth"
	}

	if realm == "" {
		realm = "Restricted"
	}

	if passwordFileLocation == "" {
		passwordFileLocation = "./htpasswd"
	}

	if sessionKey == "" || len(sessionKey) != 32 {
		log.Printf("warn: TRAUTH_SESSION_KEY is empty or has invalid length. need a 32 char key")
		log.Printf("warn: one has been generated for you, but you should change this!")
		sessionKey = string(securecookie.GenerateRandomKey(32))
	}

	log.Print("configuration information")
	log.Printf("port: %s; domain: %s; cookiePath: %s; cookieName: %s; passfile: %s",
		serverPort, domain, cookiePath, cookieName, passwordFileLocation)

	return nil
}

func main() {

	if err := parseEnv(); err != nil {
		log.Fatalf("failed parsing environment: %s", err)
	}

	log.Printf("initializing cookie keys and options...")
	store = sessions.NewCookieStore([]byte(sessionKey))
	store.Options = &sessions.Options{
		Domain:   domain,
		Path:     cookiePath,
		MaxAge:   ((60 * 60) * 24) * 365, // ((h) d) y
		HttpOnly: cookieHttpOnly,
		Secure:   cookieSecure,
	}

	// register the User type for getUser()
	gob.Register(User{})

	// read the password file
	log.Printf("reading password file at %s...\n", passwordFileLocation)
	var err error
	passwordFile, err = htpasswd.New(passwordFileLocation, htpasswd.DefaultSystems, nil)
	if err != nil {
		log.Fatalf("failed to read the password file with error: %s\n", err)
	}

	log.Printf("starting up... authenticating for domain %s on port %s", domain, serverPort)

	okHandler := http.HandlerFunc(Ok)
	http.Handle("/", check(okHandler))

	if err := http.ListenAndServe(fmt.Sprintf(":%s", serverPort), nil); err != nil {
		log.Fatalf("http server failed with error: %s\n", err)
	}
}
