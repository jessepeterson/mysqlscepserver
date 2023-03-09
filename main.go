package main

import (
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/micromdm/scep/v2/challenge"
	"github.com/micromdm/scep/v2/depot"
	scepserver "github.com/micromdm/scep/v2/server"
)

var version string

func main() {
	var (
		flDSN       = flag.String("dsn", envString("SCEP_DSN", ""), "SQL data source name (connection string)")
		flAPIKey    = flag.String("api", envString("SCEP_API_KEY", ""), "API key for challenge API endpoints")
		flChallenge = flag.String("challenge", envString("SCEP_CHALLENGE_PASSWORD", ""), "static challenge password (disables dynamic challenges")
		flListen    = flag.String("listen", envString("SCEP_HTTP_LISTEN", ":8080"), "port to listen on")
		flCAPass    = flag.String("capass", envString("SCEP_CA_PASS", ""), "passwd for the ca.key")
		flDebug     = flag.Bool("debug", envBool("SCEP_LOG_DEBUG"), "enable debug logging")
		flVersion   = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	logger := log.NewLogfmtLogger(os.Stderr)
	if !*flDebug {
		logger = level.NewFilter(logger, level.AllowInfo())
	}
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)
	lginfo := level.Info(logger)

	if *flDSN == "" || *flCAPass == "" || *flAPIKey == "" {
		fmt.Println("must supply DSN, CA pass, and API key")
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
	if *flChallenge == "" {
		signer = challenge.Middleware(mysqlDepot, signer)
	} else {
		signer = scepserver.ChallengeMiddleware(*flChallenge, signer)
	}

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

	mux := http.NewServeMux()
	mux.Handle("/scep", h)
	mux.Handle("/challenge", basicAuth(ChallengeHandlerFunc(mysqlDepot, lginfo), "api", *flAPIKey, "SCEP Challenge"))
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"` + version + `"}`))
	})

	// start http server
	errs := make(chan error, 2)
	go func() {
		lginfo.Log("transport", "http", "listen", *flListen, "msg", "listening")
		errs <- http.ListenAndServe(*flListen, mux)
	}()
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	lginfo.Log("terminated", <-errs)

}

func ChallengeHandlerFunc(store challenge.Store, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		output := &struct {
			Error     string `json:"error,omitempty"`
			Challenge string `json:"challenge,omitempty"`
		}{}
		var err error
		output.Challenge, err = store.SCEPChallenge()
		if err != nil {
			output.Error = err.Error()
			logger.Log("msg", "scep challenge", "err", err)
		}
		json, err := json.MarshalIndent(output, "", "\t")
		if err != nil {
			logger.Log("msg", "marshal json", "err", err)
		}
		w.Header().Set("Content-type", "application/json")
		_, err = w.Write(json)
		if err != nil {
			logger.Log("msg", "writing body", "err", err)
		}
	}
}

func basicAuth(next http.Handler, username, password, realm string) http.HandlerFunc {
	uBytes := []byte(username)
	pBytes := []byte(password)
	return func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(u), uBytes) != 1 || subtle.ConstantTimeCompare([]byte(p), pBytes) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
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
