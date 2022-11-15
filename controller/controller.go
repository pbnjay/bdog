package controller

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/pbnjay/bdog"
)

type Controller struct {
	Name    string
	Version string
	BaseURL string

	ReadOnly     bool
	CORSEnabled  bool
	OpenAPIRoute string

	mod     bdog.Model
	router  *httprouter.Router
	apiSpec *OpenAPI

	tokenKey   []byte
	newToken   func(string) string
	checkToken func(string) (bool, string)
}

func New(name, version string, mod bdog.Model) (*Controller, error) {
	if _, ok := mod.(bdog.Driver); !ok {
		return nil, errors.New("bdog/controller: Model does not implement Driver interface")
	}
	return &Controller{
		Name:         name,
		Version:      version,
		CORSEnabled:  true,
		ReadOnly:     false,
		OpenAPIRoute: "/openapi.json",

		mod: mod,
	}, nil
}

func (c *Controller) GenerateRoutes(extBaseURL string) http.Handler {
	c.BaseURL = extBaseURL
	c.apiSpec = NewOpenAPISpec(c.Name, c.Version, c.BaseURL)

	c.router = httprouter.New()
	if c.CORSEnabled {
		c.router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Access-Control-Request-Method") != "" {
				// Set CORS headers
				header := w.Header()
				header.Set("Access-Control-Allow-Methods", header.Get("Allow"))
				header.Set("Access-Control-Allow-Origin", "*")
			}

			// Adjust status code to 204
			w.WriteHeader(http.StatusNoContent)
		})
	}

	if c.tokenKey != nil {
		c.router.POST("/auth", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			r.ParseForm()
			who := r.Form.Get("who")
			if who == "" {
				fmt.Fprintln(w, "please provide the 'who' parameter to identify yourself.")
				return
			}
			token := c.newToken(who)

			log.Printf("New token requested for '%s' ~~ %s", who, token)
			fmt.Fprintf(w, "Token generated for '%s' please contact your administrator for it.", who)
		})
	}

	if c.OpenAPIRoute != "" {
		c.router.GET(c.OpenAPIRoute, c.apiSpec.Handler())
	}

	for _, topLevel := range c.mod.ListTableNames() {
		c.Single(topLevel)
		c.Listing(topLevel)

		rels := c.mod.ListRelatedTableNames(topLevel)
		if len(rels) > 0 {
			for _, other := range rels {
				otherTab := c.mod.GetTable(other)
				colmaps := c.mod.GetRelatedTableMappings(topLevel, other)

				for _, rights := range colmaps {
					for _, right := range rights {
						// if <right> is not the PK for <other> then this is a to-many relationshop
						if !otherTab.Key.IsEqual(right) {
							c.ListingFromSingle(topLevel, other)
						}
					}
				}
			}
		}

		if !c.ReadOnly {
			c.Insert(topLevel)
			c.Update(topLevel)
			c.Delete(topLevel)
		}
	}

	if c.tokenKey != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/auth" {
				// auth not required here
				c.router.ServeHTTP(w, r)
				return
			}

			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, "Not Authorized", http.StatusForbidden)
				return
			}
			if ok, ident := c.checkToken(h[7:]); !ok {
				http.Error(w, "Not Authorized", http.StatusForbidden)
				return
			} else {
				log.Println(ident, r.Method, r.URL.Path)
			}
			c.router.ServeHTTP(w, r)
		})
	}
	return c.router
}
