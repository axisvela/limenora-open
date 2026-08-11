# Binary Releases

Releases are built from the [LIMENORA](https://github.com/axisrobo/LIMENORA) core
repository and published in this directory.

## Download

Latest release: **not yet published** (CI wiring is in progress).

## Verify checksums

Once CI is wired, each release will publish `SHA256SUMS` alongside the archives:

```sh
sha256sum -c SHA256SUMS
```

## Supported platforms

- Windows (amd64, arm64)
- Linux (amd64, arm64)
- macOS (amd64, arm64)

Binary naming follows `limenora-{role}-{os}-{arch}` where role is
`edge`, `enterprise`, or `control`.
