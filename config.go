package trauth

import (
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	htpasswd "github.com/tg123/go-htpasswd"
)

// Config the plugin configuration.
type Config struct {
	// Required Values
	Domain    string `yaml:"domain"`
	Users     string `yaml:"users"`
	UsersFile string `yaml:"usersfile"`

	// The rules engine, used to bypass auth
	Rules []Rule `yaml:"rules"`

	// Values with internal defaults
	CookieName     string `yaml:"cookiename"`
	CookiePath     string `yaml:"cookiepath"`
	CookieSecure   bool   `yaml:"cookiesecure"`
	CookieHttpOnly bool   `yaml:"cookiehttponly"`
	CookieKey      string `yaml:"cookiekey"`
	Realm          string `yaml:"realm"`

	// Cert authentication information
	CAPath   string `yaml:"capath"`
	CertPool *x509.CertPool

	htpasswd    *htpasswd.File
	cookieStore *sessions.CookieStore
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		CookieName:     `trauth`,
		CookiePath:     `/`,
		CookieSecure:   false,
		CookieHttpOnly: false,
		Realm:          `Restricted`,
	}
}

// Validate parses configuration information.
//
// This function does a few things:
//  1. Prepare the session store
//  2. Prepare user credentials / certificates for auth.
//
// There are two types of credentials you case set. Certificates
// or credentials in htpasswd format.
//
// For htpasswd, if Users is set a buffered reader is
// configured to parse that information. If UsersFile is set,
// a file read is configured.
//
// It is an error to set both.
func (c *Config) Validate() error {

	if c.Domain == "" {
		return fmt.Errorf("a cookie domain has not been configured")
	}

	// cookiestore setup
	if c.CookieKey == "" || len(c.CookieKey) != 32 {
		c.CookieKey = string(securecookie.GenerateRandomKey(32))
	}

	c.cookieStore = sessions.NewCookieStore([]byte(c.CookieKey))
	c.cookieStore.Options = &sessions.Options{
		Domain:   c.Domain,
		Path:     c.CookiePath,
		MaxAge:   ((60 * 60) * 24) * 365, // ((h) d) y
		HttpOnly: c.CookieHttpOnly,
		Secure:   c.CookieSecure,
	}

	// process rules by compiling the provided regular expressions
	// and parsing Excluded IPNets
	for ridx, rule := range c.Rules {
		for sidx, exclude := range rule.Excludes {
			if exclude.IPNet != "" {
				_, subnet, err := net.ParseCIDR(exclude.IPNet)
				if err != nil {
					return fmt.Errorf("failed to parse source ip range '%s' for domain %s with error %s",
						exclude.IPNet, rule.Domain, err)
				}

				c.Rules[ridx].Excludes[sidx].ipNet = subnet
			}

			if exclude.Path != "" {
				rex, err := regexp.Compile(exclude.Path)
				if err != nil {
					return fmt.Errorf("failed to compile rule regex '%s' for domain %s", exclude.Path, rule.Domain)
				}

				// assign the compiled regex to the struct
				c.Rules[ridx].Excludes[sidx].regexPath = rex
			}
		}
	}

	// ca cert reading
	if c.CAPath != "" {
		cert, err := os.ReadFile(c.CAPath)
		if err != nil {
			return fmt.Errorf("failed to read ca_cert with error: %s", err)
		}

		c.CertPool = x509.NewCertPool()
		if ok := c.CertPool.AppendCertsFromPEM(cert); !ok {
			return fmt.Errorf("failed adding %s to certpool", c.CAPath)
		}
	}

	// htpasswd setup
	if c.Users != "" && c.UsersFile != "" {
		return fmt.Errorf("both users and usersfile are set for '%s'", c.Domain)
	}

	if c.Users != "" {
		credentials, err := htpasswd.NewFromReader(
			strings.NewReader(c.Users), htpasswd.DefaultSystems, nil)
		if err != nil {
			return fmt.Errorf("failed to parse users configuration for '%s' with error: %s", c.Domain, err)
		}

		c.htpasswd = credentials

		return nil
	}

	// process a usersfile
	if c.UsersFile != "" {
		credentials, err := htpasswd.New(c.UsersFile, htpasswd.DefaultSystems, nil)
		if err != nil {
			return fmt.Errorf("failed to parse users configuration for domain '%s' with error: %s",
				c.Domain, err)
		}

		c.htpasswd = credentials
	}

	return nil
}
