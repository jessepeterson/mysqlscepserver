# mysqlscepserver

`mysqlscepserver` is a small, slightly opinionated SCEP server. It uses a MySQL backend for the CA's storage. `mysqlscepserver` is largely based on the [MicroMDM SCEP](https://github.com/micromdm/scep) server.

## Usage:

Either download a release, or build from source (below). Once you have the binary, go ahead and run it:

MySQL setup and configuration is out of scope for this documentation. But suffice it to say you need a database to connect to with the tables in the `schema.sql` file.

```
$ ./mysqlscepserver-darwin-amd64 
must supply DSN, CA pass, and API key
Usage of ./mysqlscepserver-darwin-amd64:
  -api string
    	API key for challenge API endpoints
  -capass string
    	passwd for the ca.key
  -challenge string
    	static challenge password (disables dynamic challenges
  -debug
    	enable debug logging
  -dsn string
    	SQL data source name (connection string)
  -listen string
    	port to listen on (default ":8080")
  -version
    	print version and exit
```

As the error states, we need to specify a MySQL DSN, CA password, and an API key:

```
$ ./mysqlscepserver-darwin-amd64 -dsn 'scepuser:scepsecret@tcp(127.0.0.1:3306)/scepdb' -capass casecret -api apisecret
level=info ts=2021-05-29T19:01:00.755984Z caller=main.go:102 transport=http listen=:8080 msg=listening
```

The DSN is in the form that the [MySQL driver](https://github.com/go-sql-driver/mysql#dsn-data-source-name) expects.

Environment variables can be used instead of command line switches as follows:

| Environment Variable | Equivalent Switch
|--|--
| SCEP_API_KEY | -api
| SCEP_CA_PASS | -capass
| SCEP_CHALLENGE_PASSWORD | -challenge
| SCEP_LOG_DEBUG | -debug
| SCEP_DSN | -dsn
| SCEP_HTTP_LISTEN | -listen

## Challenge API

If a static challenge is not specified on the command line then the server uses to SCEP challenges to authenticate SCEP requests. The server has an API for generating one-time-use SCEP challanges:

```
$ curl -u api:apisecret http://localhost:8080/challenge && echo
{
	"challenge": "7nA8+ljk0EwHNcXCADFOCJQ4D/G9xOY9"
}
```

This challenge can then be used by a SCEP client to authenticate their SCEP request.

## Building

Have the [Go tools](https://golang.org/dl/) installed, checkout the code, then run `make`:

```
$ make
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=v0.1.0" -o mysqlscepserver-darwin-amd64 ./
```

## Docker build and run

```
make docker
docker build . --tag jessepeterson/mysqlscepserver:latest
docker run -it --rm -p 8080:8080 jessepeterson/mysqlscepserver:latest -dsn 'scepuser:scepsecret@tcp(127.0.0.1:3306)/scepdb' -capass casecret -api apisecret
```
