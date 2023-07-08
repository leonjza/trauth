package trauth

import (
	"fmt"
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

	// Values with internal defaults
	CookieName     string `yaml:"cookiename"`
	CookiePath     string `yaml:"cookiepath"`
	CookieSecure   bool   `yaml:"cookiesecure"`
	CookieHttpOnly bool   `yaml:"cookiehttponly"`
	CookieKey      string `yaml:"cookiekey"`
	Realm          string `yaml:"realm"`

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
//  2. Prepare user credentials.
//
// There are two types of credentials you case set.
// Users, and UsersFile. If Users is set, a buffered reader is
// configured to parse that information. If UsersFile is set,
// a file read is configured.
//
// It is an error to set both.
func (c *Config) Validate() error {

	if c.Domain == "" {
		return fmt.Errorf("a domain has not been configured")
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

	// htpasswd setup
	if c.Users != "" && c.UsersFile != "" {
		return fmt.Errorf("both users and usersfile is set for '%s'", c.Domain)
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

	// finally, process a usersfile
	if c.UsersFile == "" {
		return fmt.Errorf("'%s' does not have a users / usersfile configuration", c.Domain)
	}

	credentials, err := htpasswd.New(c.UsersFile, htpasswd.DefaultSystems, nil)
	if err != nil {
		return fmt.Errorf("failed to parse users configuration for domain '%s' with error: %s",
			c.Domain, err)
	}

	c.htpasswd = credentials

	return nil
}
