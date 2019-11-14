package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	htpasswd "github.com/tg123/go-htpasswd"
)

var userName string
var serverPort string
var domain string
var cookieName string
var passwordFileLocation string

var sessionCookies = make(map[string]string)
var passwordFile *htpasswd.File
var version string

func haveAuthCookie(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	if err != nil {
		log.Printf("%s cookie read error from %s: %s\n", cookieName, r.RemoteAddr, err)
		return false
	}

	for k, v := range sessionCookies {
		if c.Value == v {
			userName = k
			return true
		}
	}

	return false
}

func check(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !haveAuthCookie(r) {
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

			// add a new session
			sessionID := uuid.New().String()
			cookie := http.Cookie{
				Name:     cookieName,
				Value:    sessionID,
				Expires:  time.Now().Add(365 * 24 * time.Hour),
				Domain:   domain,
				HttpOnly: true,
				Path:     `/`,
				SameSite: 1,
			}
			http.SetCookie(w, &cookie)
			sessionCookies[user] = sessionID

			// back to where we came from
			log.Printf("authenticated %s from %s using basic auth, redirecting back to %s",
				user, r.RemoteAddr, `foo`)
			http.Redirect(w, r, `http://www.google.com/`, http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Ok returns an OK
func Ok(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello, %s!", userName)
}

func parseEnv() error {

	serverPort = os.Getenv("TRAUTH_SERVER_PORT")
	domain = os.Getenv("TRAUTH_DOMAIN")
	cookieName = os.Getenv("TRAUTH_COOKIE_NAME")
	passwordFileLocation = os.Getenv("TRAUTH_PASSWORD_FILE_LOCATION")

	if serverPort == "" {
		serverPort = "8080"
	}

	if domain == "" {
		return errors.New("empty domain")
	}

	if cookieName == "" {
		cookieName = "trauth"
	}

	if passwordFileLocation == "" {
		passwordFileLocation = "./htpasswd"
	}

	return nil
}

func main() {

	if err := parseEnv(); err != nil {
		log.Fatalf("failed parsing environment: %s", err)
	}

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
