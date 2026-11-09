# CaratCore

CaratCore is an open source tool that helps you understand and build your own
private infrastructure (a homelab) without hiding what happens on your machine.

> **Status: early development.** `doctor` works, and `install docker` can show
> its plan with `--dry-run` but cannot execute it yet. Everything else described
> in [docs/decisions.md](docs/decisions.md) is planned, not implemented.

## The idea

Before changing anything on your system, CaratCore shows what it would do and
why, and you approve it. The planned flow is:

System check -> System model -> Recommendation -> Installation plan -> Deployment

## What works today

`caratcore doctor` inspects the machine and reports whether it can run CaratCore
services. It is read-only: it never installs or modifies anything.

It checks: platform, CPU, available memory, free disk space, privileges
(root or sudo), Docker CLI / Engine / Compose, ports 53, 3000 and 3001,
internet access, and Tailscale.

Example output (Debian 13):

```
CaratCore doctor
Host: franc

[OK  ] Platform        Debian GNU/Linux 13 (trixie) amd64: supported target
[OK  ] CPU             8 logical CPU(s)
[OK  ] Memory          4.9 GiB available of 5.6 GiB
[OK  ] Disk            203.4 GiB free of 224.7 GiB on /
[INFO] Privileges      sudo is available (not verified that this user may use it)
[OK  ] Docker CLI      available (version 29.9.1)
[OK  ] Docker Engine   ready (version 29.9.1)
[OK  ] Docker Compose  available (version 5.6.0)
[OK  ] Ports           53, 3000, 3001 are free
[OK  ] Internet        download.docker.com:443 is reachable
[OK  ] Tailscale       connected

Verdict: ready
```

The command exits with code 1 if any check is FAIL.

`caratcore install docker --dry-run` shows, step by step and with the reason
for each one, how CaratCore would install Docker Engine and Docker Compose
following [Docker's official guide for Debian](https://docs.docker.com/engine/install/debian/).
Nothing is executed. Add `--json` to get the plan as JSON. If Docker and Compose
are already installed, the plan is empty. CaratCore refuses to plan (and
explains why) on unsupported platforms, when packages that conflict with
Docker's are installed, or when Docker is installed without Compose: it never
removes or changes an existing installation.

## Install

CaratCore is a single static binary: you do not need Go to run it. Download it
from the [Releases](https://github.com/BryanAguilar-Dev/CaratCore/releases)
page, check it and run it (replace `vX.Y.Z` with the version you want):

```
curl -fLO https://github.com/BryanAguilar-Dev/CaratCore/releases/download/vX.Y.Z/caratcore-linux-amd64
curl -fLO https://github.com/BryanAguilar-Dev/CaratCore/releases/download/vX.Y.Z/SHA256SUMS
sha256sum -c SHA256SUMS --ignore-missing
chmod +x caratcore-linux-amd64
./caratcore-linux-amd64 doctor
```

`sha256sum -c` must print `OK`; if not, do not run the file. There is no
`curl | sh` installer on purpose: you should see what runs on your machine.

## Build and test

Requires Go 1.26.4 or newer.

```
go vet ./...
go test ./...
go build -o caratcore ./cmd/caratcore
```

To build for a Linux server from another OS, set `GOOS=linux` and
`GOARCH=amd64` before `go build`.

## Supported platforms

Debian 13 on amd64 is the only tested target for now.

## License

Apache License 2.0. See [LICENSE](LICENSE).
