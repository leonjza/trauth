# trauth

![Docker Cloud Build Status](https://img.shields.io/docker/cloud/build/leonjza/trauth)

A simple ForwardAuth service for Traefik.

Unlike other ForwardAuth projects that enable neat OpenID / OAuth flows, `trauth` reads a simple `htpasswd` file as a credentials database, prompting via HTTP basic auth. This is perfect for private, isolated services served using Traefik needing a simple SSO solution.
