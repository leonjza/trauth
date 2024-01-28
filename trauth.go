package trauth

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
)

// Trauth is a Traefik plugin.
type Trauth struct {
	next   http.Handler
	name   string
	config *Config

	logger *log.Logger
}

// New created a new plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {

	log := NewLogger()
	log.Printf("booting trauth version: %s", version)

	// parse && validate the configuration we got
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// register our user type
	gob.Register(User{})

	// return the plugin instance
	return &Trauth{
		next:   next,
		name:   name,
		config: config,
		logger: NewLogger(),
	}, nil
}

func (t *Trauth) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	if skipViaRule(t.config.Rules, req) {
		t.logger.Printf("skipping auth due to exclusion rule matching the domain and path")
		t.next.ServeHTTP(rw, req)

		return
	}

	// continue with authentication checks
	user := getUser(t.config, req)

	if auth := user.Authenticated; !auth {
		t.logger.Printf("not authenticated. prompting for credentials")
		rw.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, t.config.Realm))
		user, pass, ok := req.BasicAuth()

		if !ok {
			http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if !t.config.htpasswd.Match(user, pass) {
			http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if err := setUser(t.config, user, rw, req); err != nil {
			t.logger.Fatalf("failed to save user session data with: %s\n", err)
		}

		t.logger.Printf("authenticated %s from %s, redirecting to %s",
			user, req.RemoteAddr, req.URL.RequestURI())

		http.Redirect(rw, req, req.URL.RequestURI(), http.StatusFound)
		return
	}

	t.next.ServeHTTP(rw, req)
}
