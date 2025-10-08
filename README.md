# mysqlscepserver

[![CI/CD](https://github.com/jessepeterson/mysqlscepserver/workflows/CI%2FCD/badge.svg)](https://github.com/jessepeterson/mysqlscepserver/actions)

`mysqlscepserver` is a small, slightly opinionated SCEP server. It uses a MySQL backend for the CA's storage. `mysqlscepserver` is largely based on the [MicroMDM SCEP](https://github.com/micromdm/scep) server.

> [!NOTE]
> The included SCEP server and CA are very basic and lack critical security-related features. You are encouraged to explore a more robust solution such as [github.com/smallstep/certificates](https://github.com/smallstep/certificates). As alluded to in [this blog post](https://micromdm.io/blog/scepping-stone/) this project's SCEP server will not likely be supported in the future.

## Getting the latest version

* Release `.zip` files containing the server should be attached to every [GitHub release](https://github.com/jessepeterson/mysqlscepserver/releases).
  * Release zips are also [published](https://github.com/jessepeterson/mysqlscepserver/actions) for every `main` branch commit.
* A Docker container is built and [published to the GHCR.io](http://ghcr.io/jessepeterson/mysqlscepserver) registry for every release.
  * `docker pull ghcr.io/jessepeterson/mysqlscepserver:latest` — `docker run ghcr.io/jessepeterson/mysqlscepserver:latest`
  * A Docker container is also published for every `main` branch commit (and tagged with `:main`)
* If you have a [Go toolchain installed](https://go.dev/doc/install) you can checkout the source and simply run `make`.

## Usage:

Either download a release, run Docker, or build from source (below). Once you have the binary, go ahead and run it:

MySQL setup and configuration is out of scope for this documentation. But suffice it to say you need a database to connect to with the tables in the `schema.sql` file.

```
$ ./mysqlscepserver-darwin-amd64
must supply DSN, CA pass, and API key
Usage of ./mysqlscepserver-darwin-amd64:
  -api string
    	API key for challenge API endpoints [SCEP_API]
  -capass string
    	passwd for the ca.key [SCEP_CAPASS]
  -challenge string
    	static challenge password (disables dynamic challenges) [SCEP_CHALLENGE]
  -debug
    	enable debug logging [SCEP_DEBUG]
  -dsn string
    	MySQL data source name (connection string) [SCEP_DSN]
  -listen string
    	port to listen on [SCEP_LISTEN] (default ":8080")
  -version
    	print version and exit
```

As the error states, we need to specify a MySQL DSN, CA password, and an API key:

```
$ ./mysqlscepserver-darwin-amd64 -dsn 'scepuser:scepsecret@tcp(127.0.0.1:3306)/scepdb' -capass casecret -api apisecret
level=info ts=2021-05-29T19:01:00.755984Z caller=main.go:102 transport=http listen=:8080 msg=listening
```

The DSN is in the form that the [MySQL driver](https://github.com/go-sql-driver/mysql#dsn-data-source-name) expects.

Environment variables can be used instead of command line switches. See the help output above, environment variables are listed in square brackets like `SCEP_DEBUG`.

## Challenge API

If a static challenge is not specified on the command line then the server uses dynamic SCEP challenges to authenticate SCEP requests. The server has an API for generating one-time-use SCEP challanges:

```
$ curl -u api:apisecret http://localhost:8080/challenge && echo
{
	"challenge": "7nA8+ljk0EwHNcXCADFOCJQ4D/G9xOY9"
}
```

This challenge can then be used by a SCEP client to authenticate the SCEP request.

## Building

Have the [Go tools](https://golang.org/dl/) installed, checkout the code, then run `make`:

```
$ make
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=v0.1.0" -o mysqlscepserver-darwin-amd64 ./
```

## Docker build and run

To manually build a Docker image from the source and run it you could do something like this. Note, per above, we also publish Docker images to GHCR.

```
make docker
docker build --tag jessepeterson/mysqlscepserver:source .
docker run -it --rm -p 8080:8080 jessepeterson/mysqlscepserver:source -dsn 'scepuser:scepsecret@tcp(127.0.0.1:3306)/scepdb' -capass casecret -api apisecret
```

