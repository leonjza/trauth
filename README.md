# trauth

🔑 A simple, cookie based Traefik middleware [plugin](https://plugins.traefik.io/plugins) for HTTP Basic Single Sign-on

`trauth` reads [htpasswd](https://httpd.apache.org/docs/2.4/programs/htpasswd.html) formatted data as a credentials database, prompting via HTTP basic auth. Once authenticated, a (configurable) cookie will be set such that any other services sharing that domain will also be authenticated.

## usage

As this is a middleware plugin, you need to do two things. Install the plugin and then configure the middleware.

### installation

A static configuration is needed. Either add it to your existing `traefik.yml` like this:

```yml
experimental:
  plugins:
    trauth:
      moduleName: github.com/leonjza/trauth
      version: 2.0.0 # or whatever is the latest version
```

Or, add it as labels to your `docker-compose.yml` (or similar).

```text
services:
  traefik:
    image: traefik
    command:
      - --experimental.plugins.trauth.moduleName=github.com/leonjza/trauth
      - --experimental.plugins.trauth.version=2.0.0
```

### configuration

As this plugin is middleware, you need to both attach the middleware to an appropriate `http.router`, as well as configure the middleware itself. Configuration will depend on how you use Traefik, but here are some examples.

The available configuration options are as follows:

| Option | Required | Default Value | Notes |
|--------|----------|---------------|-------|
| `domain` | True | | The domain name the authentication cookie will be scoped to. |
| `realm` | False | `Restricted` | A message to display when prompting for credentials. Note, not all browsers show this to users anymore.  |
| `cookiename` | False | `trauth` | A message to display when prompting for credentials. Note, not all browsers show this to users anymore. |
| `cookiepath` | False | `/` | The name of the cookie to use for authentication. |
| `cookiekey` | False | generated | The authentication key used to check cookie authenticity. **Note** See [cookiekey](#cookiekey) section below |
| `cookieksecure` | False | `false` | Use the `secure` flag when setting the authentication cookie. |
| `cookiekhttponly` | False | `false` | Use the `httponly` flag when setting the authentication cookie. |
| `users` | False if `usersfile` is set | | A htpasswd formatted list of users to accept authentication for. If `usersfile` is not set, then this value must be set. |
| `usersfile` | False if `users` is set | | A path to a htpasswd formatted file with a list of users to accept authentication for. If `users` is not set, then this value must be set. |

#### cookiekey

By default, trauth will generate a new, random value for `cookiekey` if one is not explicitly set, or is set to a value that is not 32 asci characters long. In many cases, this is ok, however, special consideration should be given to cases where this plugin is used in multiple places. By setting a static `cookiekey`, you garuantee that cookies across instances of the plugin can read the values within. If each instance of the plugin (where multiple instances will spawn if there are multiple unique definititions of the middleware) generated their own key, none will accept the cookie value set by another.

For an example, have a look a the [docker-compose.yml](docker-compose.yml) file in this repository.

#### configuration examples

As dynamic configuration:

```yml
http:
  routers:
    myservice:
      middlewares:
        - sso
    
  middlewares:
    sso:
      plugin:
        trauth:
          domain: mydomain.local
          cookiname: sso-cookie
          users: admin:$$2y$$05$$fVvJElbTaB/Cw9FevNc2keGo6sMRhY2e55..U.6zOsca3rQuuAU1e
```

As labels:

```text
traefik.http.routers.myservice.middlewares: sso
traefik.http.middlewares.sso.plugin.trauth.domain: mydomain.local
traefik.http.middlewares.sso.plugin.trauth.cookiename: sso-cookie
# *note* the double $$ here to escape a single $
traefik.http.middlewares.sso.plugin.trauth.users: admin:$$2y$$05$$fVvJElbTaB/Cw9FevNc2keGo6sMRhY2e55..U.6zOsca3rQuuAU1e
```

## development

An example `docker-compose.dev.yml` is included to show how to get this plugin up and running. No binary releases are nessesary as the stack is configured to mount this source repository in the appropriate location.

## adding users

trauth uses a basic Apache htpasswd file format. For detailed usage of `htpasswd`, please see [this](https://httpd.apache.org/docs/2.4/programs/htpasswd.html) guide.

Users can be specified using either ther `users` or `usersfile` configuration options. In both cases, the same format is needed. For the `users` option, the content of a well formatted htpasswd file can be pasted as the value of the key. For `usersfile`, the location of a file generated using the steps that follow need to be set.

To add a new user in a new `htpass` file, using the Bcrypt hashing algorithm, run:

```bash
htpasswd -Bc htpass username1
```

To add a new user to an existing `htpass` file, run:

```bash
htpasswd -B htpass username2
```
