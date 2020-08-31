package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	htpasswd "github.com/tg123/go-htpasswd"
)

var serverPort string
var domain string
var cookiePath string
var cookieName string
var passwordFileLocation string
var passwordFile *htpasswd.File
var store *sessions.CookieStore

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

		if auth := user.Authenticated; !auth {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			user, pass, ok := r.BasicAuth()

			if !ok {
				log.Printf("no basic auth creds provided from %s\n", r.RemoteAddr)
				http.Error(w, http.StatusText(401), 401)
				return
			}

			if !passwordFile.Match(user, pass) {
				log.Printf("invalid basic auth creds provided from %s\n", r.RemoteAddr)
				http.Error(w, http.StatusText(401), 401)
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
				user, r.RemoteAddr, newDestination)

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
	cookiePath = os.Getenv("TRAUTH_COOKIE_PATH")
	cookieName = os.Getenv("TRAUTH_COOKIE_NAME")
	passwordFileLocation = os.Getenv("TRAUTH_PASSWORD_FILE_LOCATION")

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

	if passwordFileLocation == "" {
		passwordFileLocation = "./htpasswd"
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
	authKeyOne := securecookie.GenerateRandomKey(64)
	encryptionKeyOne := securecookie.GenerateRandomKey(32)
	store = sessions.NewCookieStore(authKeyOne, encryptionKeyOne)
	store.Options = &sessions.Options{
		Domain:   domain,
		Path:     "/",
		MaxAge:   ((60 * 60) * 24) * 365, // ((h) d) y
		HttpOnly: true,
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
