package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/pbnjay/bdog/controller"
	"github.com/pbnjay/bdog/drivers"
)

func main() {
	apiName := flag.String("n", "", "`name` to use in OpenAPI spec")
	apiVersion := flag.String("v", "0.0.1", "semantic `version` to use for OpenAPI spec")
	extBaseURL := flag.String("b", "http://127.0.0.1:8080/", "Full external `http://address:port/` base URL where requests will be served from")

	addr := flag.String("i", ":8080", "`address:port` to listen for API requests")
	sslCert := flag.String("s", "", "TLS `certificate.pem` for serving requests")
	sslKey := flag.String("k", "", "TLS `privateKey.pem` for serving requests")
	readOnly := flag.Bool("ro", false, "do not create write/delete endpoints")
	flag.Parse()

	dbName := flag.Arg(0)
	if dbName == "" {
		fmt.Fprintln(os.Stderr, "You must provide a database to connect to!")
		os.Exit(1)
	}

	model, err := drivers.Init(dbName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to introspect database ", dbName)
		fmt.Fprintln(os.Stderr, "  Error was: ", err)
		os.Exit(2)
	}

	c, err := controller.New(*apiName, *apiVersion, model)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to create API: ", err)
		os.Exit(3)
	}
	c.ReadOnly = *readOnly
	router := c.GenerateRoutes(*extBaseURL)

	if *sslCert != "" && *sslKey != "" {
		server := &http.Server{Addr: *addr, Handler: router}
		// TODO: swap this out with ACME / letsencrypt
		server.TLSConfig.Certificates = make([]tls.Certificate, 1)
		server.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(*sslCert, *sslKey)
		if err == nil {
			fmt.Fprintln(os.Stderr, "Starting server at ", *addr)
			err = server.ListenAndServeTLS("", "")
		}
	} else {
		fmt.Fprintln(os.Stderr, "Starting server at ", *addr)
		err = http.ListenAndServe(*addr, router)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
