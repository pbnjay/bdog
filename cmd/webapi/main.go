package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/pbnjay/bdog/controller"
	"github.com/pbnjay/bdog/drivers"
)

func main() {
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

	router := httprouter.New()

	for _, topLevel := range model.ListTableNames() {
		tab := model.GetTable(topLevel)
		keypath := ":" + strings.Join(tab.Key, "/:")

		log.Printf("GET /%s/", topLevel)
		router.GET("/"+topLevel, controller.Listing(model, topLevel))

		log.Printf("GET /%s/%s", topLevel, keypath)
		router.GET("/"+topLevel+"/"+keypath, controller.Single(model, topLevel))

		rels := model.ListRelatedTableNames(topLevel)
		if len(rels) > 0 {
			for _, other := range rels {
				otherTab := model.GetTable(other)
				colmaps := model.GetRelatedTableMappings(topLevel, other)

				for _, rights := range colmaps {
					for _, right := range rights {
						// if <right> is the PK for <other> then this is a to-one relationship, ok to nest:
						if otherTab.Key.IsEqual(right) {
							log.Printf("GET /%s/%s?include=%s", topLevel, keypath, other)
						} else {
							log.Printf("GET /%s/%s/%s", topLevel, keypath, other)
							router.GET("/"+topLevel+"/"+keypath+"/"+other,
								controller.ListingFromSingle(model, topLevel, other))
						}
					}
				}
			}
		}

		if !*readOnly {
			log.Printf("POST /%s", topLevel)
			router.POST("/"+topLevel, controller.Insert(model, topLevel))

			log.Printf("PUT /%s/%s", topLevel, keypath)
			router.PUT("/"+topLevel+"/"+keypath, controller.Update(model, topLevel))

			log.Printf("DELETE /%s/%s", topLevel, keypath)
			router.DELETE("/"+topLevel+"/"+keypath, controller.Delete(model, topLevel))
		}
	}

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
