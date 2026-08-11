# Edge Gateway Example

This example shows how to run the LIMENORA Edge Gateway locally.

```sh
./limenora-edge --port 10255 --data-dir ~/.limenora/edge
```

Verify:

```sh
curl http://127.0.0.1:10255/healthz
curl http://127.0.0.1:10255/v1/health
```

Full runtime configuration is documented in the core repository's
`docs/operations.md`.
