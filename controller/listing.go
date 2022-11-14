package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/pbnjay/bdog"
)

func (c *Controller) Listing(table string) {
	drv := c.mod.(bdog.Driver)
	tab := c.mod.GetTable(table)

	route := "/" + tab.PluralName(false)
	log.Println("GET", route)
	apiList := c.apiSpec.NewHandler("GET", route)
	apiList.Summary = "List " + tab.PluralName(true)
	apiList.Parameters = append(apiList.Parameters, APIParameter{
		Name:        "_page",
		In:          "query",
		Description: "Page number to return (default=1)",
		Schema:      APISchemaType{Type: "integer"},
	}, APIParameter{
		Name:        "_perpage",
		In:          "query",
		Description: "Number of " + tab.PluralName(true) + " per page to return (default=25)",
		Schema:      APISchemaType{Type: "integer"},
	}, APIParameter{
		Name:        "_sortby",
		In:          "query",
		Description: "Field to sort the results by (default=" + strings.Join(tab.Key, ", ") + ")",
		Schema:      APISchemaType{Type: "string"},
	})

	c.router.GET(route, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if r.Method != http.MethodGet {
			basicError(w, http.StatusMethodNotAllowed)
			return
		}
		if c.CORSEnabled {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Content-Type", "application/json")

		opts := make(map[string][]string)
		for _, colname := range tab.Key {
			key := params.ByName(colname)
			opts[colname] = append(opts[colname], key)
		}

		uq := r.URL.Query()
		for varName, vals := range uq {
			if varName == "_page" || varName == "_perpage" || varName == "_sortby" {
				opts[varName] = vals
				continue
			}

			for _, col := range tab.Columns {
				if col == varName {
					opts[varName] = vals
					opts["_filters"] = append(opts["_filters"], varName)
				}
			}
		}

		data, err := drv.Listing(tab, opts)
		if err != nil {
			log.Println(err)
			if err == bdog.ErrInvalidFilter {
				basicError(w, http.StatusBadRequest)
				return
			}
			basicError(w, http.StatusInternalServerError)
			return
		}

		err = json.NewEncoder(w).Encode(data)
		if err != nil {
			log.Println(err)
			basicError(w, http.StatusInternalServerError)
			return
		}
	})
}

// ListingFromSingle exposes a list of <table2> items related to a specified <table1> item.
// Aka a one-to-many relationship (e.g. /invoices/1234/products to list Multiple Products
// bought on a single Invoice)
func (c *Controller) ListingFromSingle(table1, table2 string) {
	drv := c.mod.(bdog.Driver)
	tab1 := c.mod.GetTable(table1)
	tab2 := c.mod.GetTable(table2)
	keypath := ":" + strings.Join(tab1.Key, "/:")

	route := "/" + tab1.PluralName(false) + "/" + keypath + "/" + tab2.PluralName(false)
	log.Println("GET", route)

	apiList2 := c.apiSpec.NewHandler("GET", route)
	apiList2.Summary = "List " + tab2.PluralName(true) + " linked to a given " + tab1.SingleName(true)
	apiList2.Parameters = append(apiList2.Parameters, APIParameter{
		Name:        "_page",
		In:          "query",
		Description: "Page number to return (default=1)",
		Schema:      APISchemaType{Type: "integer"},
	}, APIParameter{
		Name:        "_perpage",
		In:          "query",
		Description: "Number of " + tab2.PluralName(true) + " per page to return (default=25)",
		Schema:      APISchemaType{Type: "integer"},
	}, APIParameter{
		Name:        "_sortby",
		In:          "query",
		Description: "Field to sort the results by (default=" + strings.Join(tab2.Key, ", ") + ")",
		Schema:      APISchemaType{Type: "string"},
	})

	c.router.GET(route, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if r.Method != http.MethodGet {
			basicError(w, http.StatusMethodNotAllowed)
			return
		}
		if c.CORSEnabled {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Content-Type", "application/json")

		key := params.ByName(tab1.Key[0])
		opts := make(map[string][]string)

		uq := r.URL.Query()
		for varName, vals := range uq {
			if varName == "_page" || varName == "_perpage" || varName == "_sortby" {
				opts[varName] = vals
				continue
			}

			for _, col := range tab2.Columns {
				if col == varName {
					opts[varName] = vals
					opts["_filters"] = append(opts["_filters"], varName)
				}
			}
		}

		c.mod.GetSubqueryMapping(tab1, tab2, key, opts)

		data, err := drv.Listing(tab2, opts)
		if err != nil {
			log.Println(err)
			if err == bdog.ErrInvalidInclude {
				basicError(w, http.StatusBadRequest)
				return
			}
			basicError(w, http.StatusInternalServerError)
			return
		}

		err = json.NewEncoder(w).Encode(data)
		if err != nil {
			log.Println(err)
			basicError(w, http.StatusInternalServerError)
			return
		}
	})
}
