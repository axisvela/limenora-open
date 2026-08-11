# LIMENORA Binaries

Prebuilt LIMENORA runtime binaries are published on the
[Releases](https://github.com/axisrobo/limenora-open/releases) page of this
repository. Each release tag (`v0.1.0`, `v1.0.0`, ...) contains archives for
Linux, macOS, and Windows on amd64 and arm64.

The binaries are built from the AGPL-3.0
[limenora](https://github.com/axisrobo/limenora) core runtime and published here
for convenience. They remain subject to the AGPL-3.0 license. The v0.1.0-v0.3.0
archives were built locally with Go 1.25.5; future releases are produced by the
CI pipeline in the core repository.

## Contents of each archive

- `limenora-edge` — LIMENORA Edge Gateway (device / agent host egress)
- `limenora-enterprise` — LIMENORA Enterprise Gateway (central ingress/egress)
- `limenora-gateway` — compatibility alias for the Edge Gateway
- `limenora-control` — LIMENORA Control Plane (routes/connectors/endpoints)

## Download

Latest release: **v0.3.0** (Wave 2 — Enterprise Ingress & Scale)

```sh
# example for linux-amd64
curl -L -o limenora.tgz \
  https://github.com/axisrobo/limenora-open/releases/download/v0.3.0/limenora-v0.3.0-linux-amd64.tar.gz
tar -xzf limenora.tgz
```

## Verify checksums

```sh
# from the Releases page, download SHA256SUMS and verify
sha256sum -c SHA256SUMS
```

## Run

Start the Edge Gateway:

```sh
./limenora-edge --port 10255 --data-dir ~/.limenora/edge
```

Start the Enterprise Gateway:

```sh
./limenora-enterprise --port 10256 --data-dir ~/.limenora/enterprise
```

Health check:

```sh
curl http://127.0.0.1:10255/healthz
curl http://127.0.0.1:10255/v1/health
```

## Building from source

To build locally from the core repository:

```sh
git clone https://github.com/axisrobo/limenora
cd limenora
go build -o limenora-edge ./cmd/client-gateway
go build -o limenora-enterprise ./cmd/server-gateway
go build -o limenora-gateway ./cmd/gateway
```

Requires Go 1.25.5+.

## Supported platforms

| OS | Architectures |
|---|---|
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64, arm64 |

## Release workflow

Pushing a `v*` tag to the [limenora](https://github.com/axisrobo/limenora) core
repository triggers `.github/workflows/release-binaries.yml` there, which builds
the runtime and publishes the binaries and checksums to this repository's
[Releases](https://github.com/axisrobo/limenora-open/releases) page.
