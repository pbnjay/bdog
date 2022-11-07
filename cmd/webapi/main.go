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

	apiSpec := controller.NewOpenAPISpec(*apiName, *apiVersion, *extBaseURL)

	router := httprouter.New()
	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Access-Control-Request-Method") != "" {
			// Set CORS headers
			header := w.Header()
			header.Set("Access-Control-Allow-Methods", header.Get("Allow"))
			header.Set("Access-Control-Allow-Origin", "*")
		}

		// Adjust status code to 204
		w.WriteHeader(http.StatusNoContent)
	})

	router.GET("/openapi.json", apiSpec.Handler())

	for _, topLevel := range model.ListTableNames() {
		tab := model.GetTable(topLevel)
		keypath := ":" + strings.Join(tab.Key, "/:")

		log.Printf("GET /%s/", topLevel)
		router.GET("/"+topLevel, controller.Listing(model, topLevel))
		apiList := apiSpec.NewHandler("GET", "/"+topLevel)
		apiList.Summary = "List " + topLevel
		apiList.Parameters = append(apiList.Parameters, controller.APIParameter{
			Name:        "_page",
			In:          "query",
			Description: "Page number to return (default=1)",
			Schema:      controller.APISchemaType{Type: "integer"},
		}, controller.APIParameter{
			Name:        "_perpage",
			In:          "query",
			Description: "Number of items per page number to return (default=25)",
			Schema:      controller.APISchemaType{Type: "integer"},
		}, controller.APIParameter{
			Name:        "_sortby",
			In:          "query",
			Description: "Field to sort the results by (default=" + keypath + ")",
			Schema:      controller.APISchemaType{Type: "string"},
		})

		log.Printf("GET /%s/%s", topLevel, keypath)
		router.GET("/"+topLevel+"/"+keypath, controller.Single(model, topLevel))
		apiGet := apiSpec.NewHandler("GET", "/"+topLevel+"/"+keypath)
		apiGet.Summary = "Get one record from " + topLevel

		rels := model.ListRelatedTableNames(topLevel)
		if len(rels) > 0 {
			relIncludes := []string{}

			for _, other := range rels {
				otherTab := model.GetTable(other)
				colmaps := model.GetRelatedTableMappings(topLevel, other)

				for _, rights := range colmaps {
					for _, right := range rights {
						// if <right> is the PK for <other> then this is a to-one relationship, ok to nest:
						if otherTab.Key.IsEqual(right) {
							log.Printf("GET /%s/%s?include=%s", topLevel, keypath, other)
							relIncludes = append(relIncludes, other)
						} else {
							log.Printf("GET /%s/%s/%s", topLevel, keypath, other)
							router.GET("/"+topLevel+"/"+keypath+"/"+other,
								controller.ListingFromSingle(model, topLevel, other))

							otherKeypath := ":" + strings.Join(otherTab.Key, "/:")
							apiList2 := apiSpec.NewHandler("GET", "/"+topLevel+"/"+keypath+"/"+other)
							apiList2.Summary = "List " + other + " records linked to a given " + topLevel + " record"
							apiList2.Parameters = append(apiList2.Parameters, controller.APIParameter{
								Name:        "_page",
								In:          "query",
								Description: "Page number to return (default=1)",
								Schema:      controller.APISchemaType{Type: "integer"},
							}, controller.APIParameter{
								Name:        "_perpage",
								In:          "query",
								Description: "Number of items per page number to return (default=25)",
								Schema:      controller.APISchemaType{Type: "integer"},
							}, controller.APIParameter{
								Name:        "_sortby",
								In:          "query",
								Description: "Field to sort the results by (default=" + otherKeypath + ")",
								Schema:      controller.APISchemaType{Type: "string"},
							})
						}
					}
				}
			}

			if len(relIncludes) > 0 {
				apiGet.Parameters = append(apiGet.Parameters, controller.APIParameter{
					Name:        "include",
					In:          "query",
					Description: "include linked records, nested in the result. available options: " + strings.Join(relIncludes, ", "),
					Schema:      controller.APISchemaType{Type: "string"},
				})
			}
		}

		if !*readOnly {
			log.Printf("POST /%s", topLevel)
			router.POST("/"+topLevel, controller.Insert(model, topLevel))
			apiPost := apiSpec.NewHandler("POST", "/"+topLevel)
			apiPost.Summary = "Create a new record in " + topLevel

			log.Printf("PUT /%s/%s", topLevel, keypath)
			router.PUT("/"+topLevel+"/"+keypath, controller.Update(model, topLevel))
			apiPut := apiSpec.NewHandler("PUT", "/"+topLevel)
			apiPut.Summary = "Update (part of) a record in " + topLevel

			log.Printf("DELETE /%s/%s", topLevel, keypath)
			router.DELETE("/"+topLevel+"/"+keypath, controller.Delete(model, topLevel))
			apiDelete := apiSpec.NewHandler("DELETE", "/"+topLevel+"/"+keypath)
			apiDelete.Summary = "Delete a record in " + topLevel
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
