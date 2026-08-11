# Quick Start

## Install

Download the latest binary from the [dist README](./dist/README.md).

## Run Edge Gateway

```sh
./limenora-edge --port 10255 --data-dir ~/.limenora/edge
```

## Run Enterprise Gateway

```sh
./limenora-enterprise --port 10256 --data-dir ~/.limenora/enterprise
```

## Verify

```sh
curl http://127.0.0.1:10255/healthz
curl http://127.0.0.1:10255/v1/health
```

See the [examples](../examples) directory for more.
