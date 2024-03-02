package trauth

import (
	"context"
	"crypto/x509"
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
		t.next.ServeHTTP(rw, req)
		return
	}

	user := getUser(t.config, req)

	if auth := user.Authenticated; !auth {
		t.logger.Printf("unauthenticated request from %s to %s%s", req.RemoteAddr, req.Host, req.URL.Path)

		// check if we can try and do mTLS
		if req.TLS != nil && len(req.TLS.PeerCertificates) > 0 {
			t.tryMTLSAuth(rw, req)
			return
		}

		// fall back to basic authentication
		t.tryBasicAuth(rw, req)
		return
	}

	t.next.ServeHTTP(rw, req)
}

// tryMTLSAuth will check for any client certificates and validate them.
func (t *Trauth) tryMTLSAuth(rw http.ResponseWriter, req *http.Request) {

	// this is an important case. if Roots for Verify() is nil, it will use the
	// system CA pool. avoid that.
	if t.config.CertPool == nil {
		return
	}

	// check each peer certificate against the configured ca
	for _, cert := range req.TLS.PeerCertificates {
		_, err := cert.Verify(x509.VerifyOptions{
			Roots: t.config.CertPool,
		})

		// an empty error implies a valid certificate
		if err == nil {
			if err := setUser(t.config, cert.Subject.CommonName, rw, req); err != nil {
				t.logger.Fatalf("failed to save user session data with: %s\n", err)
			}

			t.logger.Printf("authenticated %s from %s using mTLS", cert.Subject.CommonName, req.RemoteAddr)
			http.Redirect(rw, req, req.URL.RequestURI(), http.StatusFound)

			return
		}
	}

	http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

// tryBasicAuth with prompt for HTTP basic authentication credentials
// and read the response to determine if valid credentials were given
func (t *Trauth) tryBasicAuth(rw http.ResponseWriter, req *http.Request) {

	// ensure htpassword is configured. as this is the last auth method we support,
	// return an error http code in case we dont have an htpasswd instance (meaning there
	// are no credentials configured to test to begin with)
	if t.config.htpasswd == nil {
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

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

	t.logger.Printf("authenticated %s from %s using HTTP Basic authentication", user, req.RemoteAddr)
	http.Redirect(rw, req, req.URL.RequestURI(), http.StatusFound)
}
