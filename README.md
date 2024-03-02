# trauth

<img align="right" src="./images/logo.png" height="300" alt="beacon-pip-frame-proxy">

🔑 A simple, cookie based Traefik middleware [plugin](https://plugins.traefik.io/plugins) for HTTP Basic or mTLS Single Sign-on

`trauth` can either read [htpasswd](https://httpd.apache.org/docs/2.4/programs/htpasswd.html) formatted data as a credentials database, prompting via HTTP basic auth or validate client TLS certificates against a pre-configured CA. Once authenticated, a (configurable) cookie will be set such that any other services sharing that domain will also be authenticated.

**Note:** This plugin changed significantly in the version 1.4.x release from a ForwardAuth server to a middleware plugin. If you are interested in the ForwardAuth server then checkout the [old branch](https://github.com/leonjza/trauth/tree/old).

## usage

As this is a middleware plugin, you need to do two things to use it. Install the plugin and then configure the middleware.

### installation

A static configuration is needed. Either add it to your existing `traefik.yml` like this:

```yml
experimental:
  plugins:
    trauth:
      moduleName: github.com/leonjza/trauth
      version: v1.5.0 # or whatever the latest version is
```

Or, add it as labels to your `docker-compose.yml` (or similar).

```text
services:
  traefik:
    image: traefik
    command:
      - --experimental.plugins.trauth.moduleName=github.com/leonjza/trauth
      - --experimental.plugins.trauth.version=v1.5.0
```

### authentication types

It is possible to use both mTLS and HTTP Basic Authentication at the same time. In that case, trauth will prefer mTLS over basic authentication, but if a connection came it that does not have a client certificate (or isn't TLS to begin with), it will fall back to Basic Authentication.

If none of the authentication methods are configured, it will not be possible to authenticate, meaning a webservice protected with trauth will only response with HTTP 401's.

For configuration examples refer to the docker-compose.yml files in this repo. For more information about the available configuration options see the [#configuration](#configuration) section below.

#### mtls

In the case of mTLS, a client's peer certificate will be validated against a configured Certificate Authority (CA). Configuring mTLS authentication requires two things:

- A CA needs to be configured as a path to a PEM encoded file using the `capath` configuration value.
- The relevant Traefik [TLS options](https://doc.traefik.io/traefik/https/tls/#client-authentication-mtls) need to be set on the relevant service router.

Regardless if you use docker labels or a dynamic configuration, you need to specify the relevant TLS options as a seperate file. An example can be seen in the [tls.yml](./tls.yml) file.

With mTLS configured, provision the client certificate as needed. An example `curl` request that includes a client certificate to authenticate to a `trauth` protected service, accepting a cookie to include in a redirection would be:

```bash
curl -kvL -b cookies.txt --cert ca/user1/user1.pem https://whoami-3.dev.local/
```

#### basic auth

In the case of HTTP Basic Authentication, trauth needs to have an `httpasswd` formatted user database configured. That can be done using either the `users` option to specify them inline, or via `usersfile` to set a path containing the user database.

An example `curl` request that includes the credentials to authenticate to an HTTP Basic Authentication protected service would be:

```bash
curl -kvL -b cookies.txt --user admin:password http://whoami-2.dev.local/
```

### configuration

As this plugin is middleware, you need to both attach the middleware to an appropriate `http.router`, as well as configure the middleware itself. Configuration will depend on how you use Traefik, but here are some examples.

The available configuration options are as follows:

| Option | Required | Default Value | Notes |
|--------|----------|---------------|-------|
| `domain` | True | | The domain name the authentication cookie will be scoped to. |
| `realm` | False | `Restricted` | A message to display when prompting for credentials. Note, not all browsers show this to users anymore.  |
| `capath` | False |  | A path to a PEM encoded Certificate Authority to validate client provided certificates against. |
| `cookiename` | False | `trauth` | A message to display when prompting for credentials. Note, not all browsers show this to users anymore. |
| `cookiepath` | False | `/` | The name of the cookie to use for authentication. |
| `cookiekey` | False | generated | The authentication key used to check cookie authenticity. **Note** See [cookiekey](#cookiekey) section below |
| `cookiesecure` | False | `false` | Use the `secure` flag when setting the authentication cookie. |
| `cookiehttponly` | False | `false` | Use the `httponly` flag when setting the authentication cookie. |
| `users` | False | | A htpasswd formatted list of users to accept authentication for. If `usersfile` is not set, then this value must be set. |
| `usersfile` | False | | A path to a htpasswd formatted file with a list of users to accept authentication for. If `users` is not set, then this value must be set. |
| `rules` | False | | A rules object that defines hostnames and paths where authentication requirements are skipped |

#### cookiekey

By default, trauth will generate a new, random value for `cookiekey` if one is not explicitly set, or is set to a value that is not 32 ascii characters long. In many cases, this is ok, however, special consideration should be given to cases where this plugin is used in multiple places. By setting a static `cookiekey`, you garuantee that cookies across instances of the plugin can read the values within. If each instance of the plugin (where multiple instances will spawn if there are multiple unique definititions of the middleware) generated their own key, none will accept the cookie value set by another.

For an example, have a look a the [docker-compose.yml](docker-compose.yml) file in this repository.

#### rules

Rules are optional and useful if you have situations where trauth should not block and require a valid authentication session. Rules define conditions for when authentication is not required. For example, based on a request to a specific path, or a request coming from a specific source IP network range.

In the case of path exclusions, this could be used for web services that should be authenticated by default, but some parts of the service should be available without authentication (or at least, not blocked by this middleware as it may have its own authentication you want to use instead). For example, say a web application exposes an API, and you want to have that API (or parts of it), available externally.

In the case of IP network range exclusions, this could be used if a trusted network may use a web service without extra authentication, but everyone else should provide an identity first.

Rules have two configuration options. A domain, and the relevant excludes (paths or IP networks). For some examples, have a look a the [docker-compose.dev.yml](docker-compose.dev.yml) file in this repository.

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
          cookiename: sso-cookie
          users: admin:$$2y$$05$$fVvJElbTaB/Cw9FevNc2keGo6sMRhY2e55..U.6zOsca3rQuuAU1e
          # for mTLS (/ca/ca.pem needs to be readable to the traefik service in case of docker)
          capath: /ca/ca.pem
          rules:
          - domain: service.mydomain.local
            path:
            - ^/api/v1/.*$
            - ^/api/v2/.*$
          - domain: admin.mydomain.local
            ipnet:
            - 10.0.0.0/24
```

As labels:

```text
traefik.http.routers.myservice.middlewares: sso
traefik.http.middlewares.sso.plugin.trauth.domain: mydomain.local
traefik.http.middlewares.sso.plugin.trauth.cookiename: sso-cookie
traefik.http.middlewares.sso.plugin.trauth.capath: /ca/ca.pem
# optional rules
traefik.http.middlewares.sso.plugin.trauth.rules[0].domain: service.mydomain.local
traefik.http.middlewares.sso.plugin.trauth.rules[0].excludes[0].path: ^/api/v1/.*$
traefik.http.middlewares.sso.plugin.trauth.rules[0].excludes[1].path: ^/api/v2/.*$
traefik.http.middlewares.sso.plugin.trauth.rules[1].domain: admin.mydomain.local
traefik.http.middlewares.sso.plugin.trauth.rules[1].excludes[0].path: 10.0.0.0/24
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
