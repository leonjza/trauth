# trauth

![Docker Cloud Build Status](https://img.shields.io/docker/cloud/build/leonjza/trauth)

A simple [ForwardAuth](https://docs.traefik.io/middlewares/forwardauth/) service for Traefik.

Unlike other ForwardAuth projects that enable neat OpenID / OAuth flows, `trauth` reads a simple `htpasswd` file as a credentials database, prompting via HTTP basic auth. This is perfect for private, isolated services served using Traefik needing a simple SSO solution.

## usage

An example `docker-compose.yml` is included to show how to get it up and running. It assumes that `htpass` is mounted externally.

Of course, you could compile from source or download from the [releases](https://github.com/leonjza/trauth/releases) page and run outside of docker too.

## setup

Depending on your setup, a few environment variables must be configured. For a docker-compose setup you would need `TRAUTH_DOMAIN` and `TRAUTH_PASSWORD_FILE_LOCATION` at the very least.

```yml
environment:
    - TRAUTH_DOMAIN=yourdomain.local
    - TRAUTH_PASSWORD_FILE_LOCATION=/config/htpass
```

Other variables also exist. Those are:

* `TRAUTH_SESSION_KEY` - The authentication key used to validate cookie values. This value must be a 32 character, random string. Not setting this value (or using a value of the wrong size), will result in trauth generating a random key to use. A random value means everytime trauth starts, you'd need to re-authenticate. (Defaults to random value)
* `TRAUTH_SERVER_PORT` - The port the server should listen on. (Defaults to 8080)
* `TRAUTH_DOMAIN` - The domain trauth should set the sso cookie for. This is usually scoped for the parent domain.
* `TRAUTH_COOKIE_PATH` - The path used for the sso cookie. (Defaults to `/`)
* `TRAUTH_COOKIE_NAME` - The name of the sso cookie. (Defaults to `trauth`)
* `TRAUTH_PASSWORD_FILE_LOCATION` - The location for the `htpasswd` file. (Defaults to `./htpass`)

## enabling for Traefik web services

To use it in Traefik you need to define a new middleware telling Traefik where the auth server is. For example:

```text
- "traefik.http.middlewares.trauth.forwardauth.address=http://trauth.web:8080/"
```

Next, you simply need to add the middleware label to web services that should make use of it. For example:

```text
- "traefik.http.routers.netdata.middlewares=trauth"
```

## adding users

trauth uses a basic Apache htpasswd file. For detailed usage of `htpasswd`, please see [this](https://httpd.apache.org/docs/2.4/programs/htpasswd.html) guide.

To add a new user in a new `htpass` file, using the Bcrypt hashing algorithm, run:

```bash
htpasswd -Bc htpass username1
```

To add a new user to an existing `htpass` file, run:

```bash
htpasswd -B htpass username2
```
