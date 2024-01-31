# modproxy

> modproxy directs `go get` from one location to another.

## Configuration

The behaviour of modproxy can be configured using the following environment variables:

- `HOST_PATTERN`: Specifies the pattern for host matching. Defaults to "go.loafoe.dev".
- `HOST_REPLACEMENT`: Determines the replacement for the host. Defaults to "github.com".
- `PATH_PATTERN`: Sets the pattern for path matching. Defaults to "/".
- `PATH_REPLACEMENT`: Defines the replacement for the path. Defaults to "/epiccoolguy/go-".

## Run locally

```sh
FUNCTION_TARGET=ModProxy LOCAL_ONLY=true go run cmd/main.go
```

- `FUNCTION_TARGET`: Specifies the name of the function to be executed when the server is started.
- `LOCAL_ONLY`: When set to true, the server listens only on 127.0.0.1 (localhost), restricting access to the local machine. This is useful for local testing, avoiding firewall warnings, and preventing external access to the server during development or testing phases. If not set, listen on all interfaces.

Confirm the url is correctly being rewritten:

```sh
curl -H 'Host: go.loafoe.dev' localhost:8080/modproxy
# Output: <html><head><meta name="go-import" content="go.loafoe.dev/modproxy git https://github.com/epiccoolguy/go-modproxy"></head><body></body></html>
```

## Run using `pack` and Docker

```sh
pack build \
  --builder gcr.io/buildpacks/builder:v1 \
  --env GOOGLE_FUNCTION_SIGNATURE_TYPE=http \
  --env GOOGLE_FUNCTION_TARGET=ModProxy \
  go-modproxy
```

- `GOOGLE_FUNCTION_SIGNATURE_TYPE`: Specifies the type of function signature the application uses.
- `GOOGLE_FUNCTION_TARGET`: Specifies the name of the function to execute in the application.

Run the built image:

```sh
docker run --rm -p 8080:8080 go-modproxy
```

Confirm the url is correctly being rewritten:

```sh
curl -H 'Host: go.loafoe.dev' localhost:8080/modproxy
# Output: <html><head><meta name="go-import" content="go.loafoe.dev/modproxy git https://github.com/epiccoolguy/go-modproxy"></head><body></body></html>
```

## Run using Google Cloud Platform

See [gcp.sh](./gcp.sh)
