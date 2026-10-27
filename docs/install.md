# Installation

There are several ways of installing Metrics Usage:

- [Using the precompiled binaries](#precompiled-binaries)
- [Using a container image](#container-images)
- [Building it from source](#building-from-source)

Whatever the method you choose, Metrics Usage needs a [configuration](./configuration.md) to know which collectors to
run. You can find an example in the [dev folder](https://github.com/perses/metrics-usage/blob/main/dev/config.yaml) of
the repository.

## Precompiled binaries

Download the precompiled binaries from the [GitHub releases page](https://github.com/perses/metrics-usage/releases).
It is recommended to use the latest release available.

Archives are available for Linux, macOS and Windows, on the `amd64` and `arm64` architectures (and `armv6` for Linux).

For example, on Linux `amd64`:

```bash
VERSION=0.7.0
curl -LO "https://github.com/perses/metrics-usage/releases/download/v${VERSION}/metrics-usage_${VERSION}_linux_amd64.tar.gz"
tar -xzf "metrics-usage_${VERSION}_linux_amd64.tar.gz"
./metrics-usage --config=./config.yaml
```

## Container images

Container images are available on [Docker Hub](https://hub.docker.com/r/persesdev/metrics-usage).

Start Metrics Usage with your configuration file using one of the following commands depending on your tooling:

```bash
# podman command
podman run --name metrics-usage --rm -p 127.0.0.1:8080:8080 \
  -v "$(pwd)/config.yaml:/etc/metrics-usage/config.yaml:ro" \
  persesdev/metrics-usage --config=/etc/metrics-usage/config.yaml
```

```bash
# docker command
docker run --name metrics-usage --rm -p 127.0.0.1:8080:8080 \
  -v "$(pwd)/config.yaml:/etc/metrics-usage/config.yaml:ro" \
  persesdev/metrics-usage --config=/etc/metrics-usage/config.yaml
```

The details in this command are that we give the container a referenceable name (`--name metrics-usage`),
automatically remove the container when it stops (`--rm`), map the local machine port 8080 to the container port 8080
(`-p 127.0.0.1:8080:8080`), mount the configuration file in the container (`-v`) and tell Metrics Usage where to find
it (`--config`).

As an alternative to the configuration file, you can also configure Metrics Usage with
[environment variables](./configuration.md#environment-variables):

```bash
docker run --name metrics-usage --rm -p 127.0.0.1:8080:8080 \
  -e METRICS_USAGE_METRIC_COLLECTOR_ENABLE=true \
  -e METRICS_USAGE_METRIC_COLLECTOR_HTTP_CLIENT_URL=https://prometheus.demo.prometheus.io \
  persesdev/metrics-usage
```

Once started, the API is available at <http://localhost:8080/api/v1/metrics>.

### Tags

We are publishing a set of tags to match different requirements:

1. The images can run either on the **amd64** or on the **arm64** platform.
2. There are two types of image: `distroless` and `distroless-debug`.
   The first one is meant to be used in production as it doesn't contain anything except the Metrics Usage binary.
   The second one contains a shell and should be used to debug any issue you might encounter with your container.
3. Each release is available with the full version (`v0.7.0`) or only the minor version (`v0.7`), with or without the
   distribution suffix (`v0.7.0-distroless`, `v0.7.0-distroless-debug`). Without suffix, the image is the `distroless`
   one, and the suffix `-debug` is an alias for `-distroless-debug`.
4. The tag `latest` is an alias for the `distroless` distribution of the latest release.
5. The tag `latest-debug` is an alias for the `distroless-debug` distribution of the latest release.

Examples:

```bash
docker pull persesdev/metrics-usage:v0.7.0
docker pull persesdev/metrics-usage:v0.7-debug
docker pull persesdev/metrics-usage:latest-debug
```

## Building from source

To build Metrics Usage from source, you need [Go](https://go.dev/doc/install) in the version defined in the
[`go.mod`](https://github.com/perses/metrics-usage/blob/main/go.mod) file of the repository.

Start by cloning the repository:

```bash
git clone https://github.com/perses/metrics-usage.git
cd metrics-usage
```

Then build Metrics Usage with:

```bash
make build
```

The binary is created in the folder `bin`. You can run it with:

```bash
./bin/metrics-usage --config=./dev/config.yaml
```

## What's next

- [Choose how to deploy Metrics Usage](./deployment.md)
- [Configure Metrics Usage](./configuration.md)
