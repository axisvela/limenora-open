# Enterprise Gateway Example

This example shows how to run the LIMENORA Enterprise Gateway (central ingress/egress).

```sh
./limenora-enterprise --port 10256 --data-dir ~/.limenora/enterprise
```

With policy resolution enabled:

```sh
DATABASE_URL=postgres://... SECRET_ENCRYPTION_KEY=... ./limenora-enterprise --port 10256
```

See the core repository's `docs/operations.md` for environment variable
reference and the internal LIMENORA-ee product plan for enterprise topology.
