# CaratCore

CaratCore is an open source tool that helps you understand and build your own
private infrastructure (a homelab) without hiding what happens on your machine.

> **Status: early development.** `doctor`, installing Docker and deploying
> AdGuard Home and Uptime Kuma work on Debian 13 amd64. Updating and removing
> services, backups and rollback do not exist yet; see
> [docs/decisions.md](docs/decisions.md) for what is planned.

## The idea

Before changing anything on your system, CaratCore shows what it would do and
why, and you approve it. The planned flow is:

System check -> System model -> Recommendation -> Installation plan -> Deployment

## What works today

`caratcore doctor` inspects the machine and reports whether it can run CaratCore
services. It is read-only: it never installs or modifies anything.

It checks: platform, CPU, available memory, free disk space, privileges
(root or sudo), Docker CLI / Engine / Compose, ports 53, 3000 and 3001,
internet access, and Tailscale. Ports used by a service CaratCore deployed are
reported as OK.

Example output (Debian 13):

```
CaratCore doctor
Host: franc

[OK  ] Platform        Debian GNU/Linux 13 (trixie) amd64: supported target
[OK  ] CPU             8 logical CPU(s)
[OK  ] Memory          4.9 GiB available of 5.6 GiB
[OK  ] Disk            202.7 GiB free of 224.7 GiB on /
[INFO] Privileges      sudo is available (not verified that this user may use it)
[OK  ] Docker CLI      available (version 29.9.3)
[INFO] Docker Engine   not checked: this user cannot use Docker without sudo (CaratCore uses Docker with sudo)
[OK  ] Docker Compose  available (version 5.7.0)
[OK  ] Ports           53, 3000, 3001 are free
[OK  ] Internet        download.docker.com:443 is reachable
[OK  ] Tailscale       connected

Verdict: ready
```

The command exits with code 1 if any check is FAIL.

`caratcore install docker --dry-run` shows, step by step and with the reason
for each one, how CaratCore would install Docker Engine and Docker Compose
following [Docker's official guide for Debian](https://docs.docker.com/engine/install/debian/).
Nothing is executed. Add `--json` to get the plan as JSON. Unlike the guide,
CaratCore does not download Docker's signing key: the key is built into the
binary and a test checks its fingerprint against the one Docker publishes.
If Docker and Compose are already installed, the plan is empty. CaratCore refuses to plan (and
explains why) on unsupported platforms, when packages that conflict with
Docker's are installed, or when Docker is installed without Compose: it never
removes or changes an existing installation.

`caratcore install docker` (without `--dry-run`) shows the same plan and asks
you to type `yes` before running it. Steps that need root run through `sudo`
(sudo asks for your password; CaratCore never sees it), or directly if you
run CaratCore as root. Each command is printed
before it runs. If a step fails, CaratCore stops, tells you which steps were
done and which were not, and undoes nothing; fix the problem and run the
command again. CaratCore never adds your user to the `docker` group, because
that is equivalent to giving it root: use `sudo docker ...`.

`caratcore install adguard-home` and `caratcore install uptime-kuma` deploy
those services with Docker Compose, with the same plan, `--dry-run` and
approval as Docker. Each service lives in `/opt/caratcore/<service>/`: its
`compose.yaml` (readable, with a pinned image version) and its data folders.
CaratCore refuses to deploy when Docker is missing, when a port is used by
another program, or when you edited the `compose.yaml` it wrote. After
installing, it tells you what to do next (for example, AdGuard Home's setup
wizard on port 3000).

| Service | What it does | Ports |
|---|---|---|
| AdGuard Home | blocks ads and trackers for the devices that use it as DNS | 53 (DNS), 3000 (web) |
| Uptime Kuma | checks that your services are up and alerts you | 3001 (web) |

## Network exposure

The ports in the table above are open to every network the machine is on,
**even if you use a firewall such as ufw**. Docker adds its own firewall rules
for published ports, and they are applied before ufw's
([Docker's documentation](https://docs.docker.com/engine/network/packet-filtering-firewalls/)).

CaratCore is meant for a home server behind a router. In that case, devices
on your local network and on your Tailscale network can reach the services,
and the internet cannot. To keep it that way:

- Do not forward ports 53, 3000 or 3001 on your router.
- Do not deploy these services on a machine with a public IP address (for
  example, a VPS) unless you filter the traffic yourself, for instance in
  Docker's `DOCKER-USER` chain. Otherwise anyone could use your DNS server.

## Install

CaratCore is a single static binary: you do not need Go or anything else to
run it. Everything it installs (Docker, Compose, the services) is part of the
plans it shows you.

### From a fresh Debian 13

**1. Make sure you can use sudo.** If you set a root password while installing
Debian, the installer did not install sudo. Either run CaratCore as root, or
set up sudo once (as root, replacing `<user>` with your user name):

```
su -
apt install sudo
adduser <user> sudo
exit
```

Then log out and log in again, so your session joins the `sudo` group.

**2. Download CaratCore and check it.** Replace `vX.Y.Z` with the version from
the [Releases](https://github.com/BryanAguilar-Dev/CaratCore/releases) page.
`wget` comes with Debian (if it is missing: `sudo apt install wget`).

```
wget https://github.com/BryanAguilar-Dev/CaratCore/releases/download/vX.Y.Z/caratcore-linux-amd64
wget https://github.com/BryanAguilar-Dev/CaratCore/releases/download/vX.Y.Z/SHA256SUMS
sha256sum -c SHA256SUMS
chmod +x caratcore-linux-amd64
```

`sha256sum -c` must print `caratcore-linux-amd64: OK`; if not, do not run the
file. There is no `curl | sh` installer on purpose: you should see what runs
on your machine.

**3. Check the machine, install Docker, deploy the services.** Each `install`
shows its plan and waits for you to type `yes`. Add `--dry-run` to only see
the plan.

```
./caratcore-linux-amd64 doctor
./caratcore-linux-amd64 install docker
./caratcore-linux-amd64 install adguard-home
./caratcore-linux-amd64 install uptime-kuma
```

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
