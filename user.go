package trauth

import "net/http"

// User holds a users session information.
type User struct {
	Username      string
	Authenticated bool
}

const cookieKey = `user`

func getUser(config *Config, req *http.Request) User {

	session, _ := config.cookieStore.Get(req, config.CookieName)
	val := session.Values[cookieKey]
	user, ok := val.(User)
	if !ok {
		return User{Authenticated: false}
	}

	return user
}

func setUser(config *Config, user string, rw http.ResponseWriter, req *http.Request) error {

	session, _ := config.cookieStore.Get(req, config.CookieName)
	session.Values[cookieKey] = &User{
		Username:      user,
		Authenticated: true,
	}

	if err := config.cookieStore.Save(req, rw, session); err != nil {
		return err
	}

	return nil
}
