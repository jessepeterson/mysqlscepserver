package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/micromdm/scep/v2/depot"
	scepserver "github.com/micromdm/scep/v2/server"
)

func main() {
	var (
		flDSN = flag.String("dsn", "", "SQL data source name (connection string)")
		// flAPIKey = flag.String("api", "", "API key for challenge API endpoints")
		flListen = flag.String("listen", envString("SCEP_HTTP_LISTEN", ":8080"), "port to listen on")
		flCAPass = flag.String("capass", envString("SCEP_CA_PASS", ""), "passwd for the ca.key")
		flDebug  = flag.Bool("debug", envBool("SCEP_LOG_DEBUG"), "enable debug logging")
	)
	flag.Parse()

	logger := log.NewLogfmtLogger(os.Stderr)
	if !*flDebug {
		logger = level.NewFilter(logger, level.AllowInfo())
	}
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)
	lginfo := level.Info(logger)

	if *flDSN == "" {
		fmt.Println("must supply DSN")
		flag.Usage()
		os.Exit(1)
	}

	if *flCAPass == "" {
		fmt.Println("must supply CA pass")
		flag.Usage()
		os.Exit(1)
	}

	mysqlDepot, err := NewMySQLDepot(*flDSN)
	if err != nil {
		lginfo.Log("err", err)
		os.Exit(1)
	}

	crt, key, err := mysqlDepot.CreateOrLoadCA([]byte(*flCAPass), 10, "ca", "scep", "US")
	if err != nil {
		lginfo.Log("err", err)
		os.Exit(1)
	}

	var signer scepserver.CSRSigner = depot.NewSigner(
		mysqlDepot,
		depot.WithAllowRenewalDays(0),
		depot.WithValidityDays(3650),
		depot.WithCAPass(*flCAPass),
	)

	svc, err := scepserver.NewService(crt, key, signer, scepserver.WithLogger(logger))
	if err != nil {
		lginfo.Log("err", err)
		os.Exit(1)
	}

	var h http.Handler // http handler
	{
		e := scepserver.MakeServerEndpoints(svc)
		e.GetEndpoint = scepserver.EndpointLoggingMiddleware(lginfo)(e.GetEndpoint)
		e.PostEndpoint = scepserver.EndpointLoggingMiddleware(lginfo)(e.PostEndpoint)
		h = scepserver.MakeHTTPHandler(e, svc, log.With(lginfo, "component", "http"))
	}

	// start http server
	errs := make(chan error, 2)
	go func() {
		lginfo.Log("transport", "http", "liste", *flListen, "msg", "listening")
		errs <- http.ListenAndServe(*flListen, h)
	}()
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	lginfo.Log("terminated", <-errs)

}

func envString(key, def string) string {
	if env := os.Getenv(key); env != "" {
		return env
	}
	return def
}

func envBool(key string) bool {
	if env := os.Getenv(key); env == "true" {
		return true
	}
	return false
}
