# CaratCore

CaratCore is an open source platform for building and managing personal infrastructure. This repository currently contains an early Go prototype.

## Current prototype

Run the diagnostic from the repository root:

```sh
go run ./cmd/caratcore doctor
```

It reports the host name, operating system, Linux distribution and version when available, architecture, whether the platform matches the initial Debian 13 `amd64` target, whether the Docker CLI is available, and whether the Docker Engine responds. The `help` argument displays usage. The diagnostic does not install Docker or create, start, or modify containers.

## First validation host

The first target used for validation is Franc, a headless Debian 13 (Trixie) server with an `amd64` architecture. A Linux `amd64` binary built from this source ran there and reported:

```text
CaratCore
Node: franc
OS: linux
Distribution: debian 13
Architecture: amd64
Platform compatibility: supported (Debian 13 amd64)
Docker CLI: not found
Docker Engine: not checked
```

This confirms host and platform discovery on Franc. Docker installation and container management have not been implemented yet.

## Development

The module currently declares Go 1.26.4 or newer. Build or run it from the repository root so Go can find `go.mod`.
