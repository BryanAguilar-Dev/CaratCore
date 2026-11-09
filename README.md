# CaratCore

CaratCore is an open source tool that helps you understand and build your own
private infrastructure (a homelab) without hiding what happens on your machine.

> **Status: early development.** Only the read-only `doctor` command exists.
> Everything else described in [docs/decisions.md](docs/decisions.md) is planned,
> not implemented.

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
