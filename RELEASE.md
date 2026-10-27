# Releases

This page describes how to release a new version of metrics-usage.

## Versioning

We use [Semantic Versioning](https://semver.org/).

The version is derived from the git tag (`git describe --tags`). There is no version file to update.

New features and bug fixes are merged into the `main` branch. We don't maintain older versions. If an urgent fix
can't wait for the next minor release, we only patch the latest version.

## What the CI does

The [`ci`](.github/workflows/ci.yml) workflow runs `make cross-build` on every pull request and on every push to
`main`, `release/*` and `snapshot/*`. This command generates the GoReleaser config and runs GoReleaser in snapshot
mode. It builds every binary and Docker image without publishing anything.

When you push a tag matching `v*`, the workflow also runs `make cross-release`, which:

- builds the binaries for Linux, macOS and Windows (amd64, arm and arm64) and publishes them in `.tar.gz` archives in
  a [GitHub release](https://github.com/perses/metrics-usage/releases). GitHub generates the changelog from the
  merged pull requests.
- builds the multi-arch (`linux/amd64`, `linux/arm64`) Docker images and pushes them to `docker.io/persesdev/metrics-usage`
  and `quay.io/persesdev/metrics-usage` with these tags:
  - `latest`, `vX.Y.Z`, `vX.Y`, `vX.Y.Z-distroless`, `vX.Y-distroless`
  - the same tags with the `-debug` suffix for the debug image (see [distroless-debug.Dockerfile](distroless-debug.Dockerfile))

The Docker images are defined in [scripts/generate-goreleaser](scripts/generate-goreleaser/generate-goreleaser.go),
and the rest of the GoReleaser configuration is in
[.goreleaser.base.yaml](scripts/generate-goreleaser/.goreleaser.base.yaml).

## How to cut a release

### 1. Tag the release

Make sure the latest `ci` run on `main` succeeded, then push the tag from `main`:

```bash
git checkout main
git pull
git tag -s v<X.Y.Z> -m "v<X.Y.Z>"
git push origin v<X.Y.Z>
```

Signing the tag with a GPG key is appreciated. If you can't sign it, replace `-s` with `-a`.

For a pre-release, use a version like `v<X.Y.Z>-rc.0`. GoReleaser marks it as a pre-release on GitHub
(`prerelease: auto`).

> [!WARNING]
> The `latest` Docker tag is pushed for every tag, including pre-releases and patches of older versions.

### 2. Check the release

1. Wait for the `ci` workflow of the tag to succeed in
   [GitHub Actions](https://github.com/perses/metrics-usage/actions/workflows/ci.yml).
2. Check that the [GitHub release](https://github.com/perses/metrics-usage/releases) has the archives and that the
   changelog looks right. Edit the release notes if needed.
3. Check that the Docker images are available on [Docker Hub](https://hub.docker.com/r/persesdev/metrics-usage)
   and [Quay.io](https://quay.io/repository/persesdev/metrics-usage).

### Patching a release

To patch an older minor version, create a `release/v<X.Y>` branch from the `v<X.Y.Z>` tag. Cherry-pick the fixes
into it, then tag the patch version from that branch the same way as above.
