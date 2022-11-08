package controller

import (
	"errors"
	"net/http"

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

	if c.OpenAPIRoute != "" {
		c.router.GET(c.OpenAPIRoute, c.apiSpec.Handler())
	}

	for _, topLevel := range c.mod.ListTableNames() {
		c.Listing(topLevel)
		c.Single(topLevel)

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
	return c.router
}
