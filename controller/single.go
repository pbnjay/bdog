package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/pbnjay/bdog"
)

func (c *Controller) Single(table string) {
	tab := c.mod.GetTable(table)
	drv := tab.Driver
	keypath := ":" + strings.Join(tab.Key, "/:")

	route := "/" + tab.PluralName(false) + "/" + keypath
	log.Println("GET", route)
	apiGet := c.apiSpec.NewHandler("GET", route)
	apiGet.Summary = "Get details for a given " + tab.SingleName(true)
	example, err := drv.Get(tab, nil)
	if err == nil {
		apiGet.AddExampleResponse("The requested "+tab.SingleName(true)+" details", example)
	}

	// map from the "include" singular label to the table name
	includeMap := make(map[string]string)
	relIncludes := []string{}
	rels := c.mod.ListRelatedTableNames(table)
	if len(rels) > 0 {

		for _, other := range rels {
			otherTab := c.mod.GetTable(other)
			colmaps := c.mod.GetRelatedTableMappings(table, other)

			for _, rights := range colmaps {
				for _, right := range rights {
					// if <right> is the PK for <other> then this is a to-one relationship, ok to nest:
					if otherTab.Key.IsEqual(right) {
						oname := otherTab.SingleName(false)
						log.Printf("GET %s?include=%s", route, oname)
						relIncludes = append(relIncludes, oname)

						includeMap[oname] = otherTab.Name
					}
				}
			}
		}

		if len(relIncludes) > 0 {
			apiGet.Parameters = append(apiGet.Parameters, APIParameter{
				Name:        "include",
				In:          "query",
				Description: "include linked records, nested in the result. available options: " + strings.Join(relIncludes, ", "),
				Schema:      APISchemaType{Type: "string"},
			})
		}
	}

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
		if len(uq["include"]) != 0 {
			for _, incName := range uq["include"] {
				incTabName, validInclude := includeMap[incName]
				if !validInclude {
					fmt.Fprintf(w, "Invalid 'include' given. Available options are: "+strings.Join(relIncludes, ", "))
					basicError(w, http.StatusBadRequest)
					return
				}
				opts["_nest"] = append(opts["_nest"], incTabName)
			}
		}

		data, err := drv.Get(tab, opts)
		if err == bdog.ErrNotFound && len(tab.Key) == 1 && len(tab.UniqueColumns) > 0 {
			// secondary check for unique key as the lookup
			qval := opts[tab.Key[0]]
			nested, hasNest := opts["_nest"]
			for _, colname := range tab.UniqueColumns {
				opts = make(map[string][]string)
				opts[colname] = qval
				if hasNest {
					opts["_nest"] = nested
				}
				data, err = drv.Get(tab, opts)
				if err == nil {
					break
				}
			}
		}
		if err != nil {
			log.Println(err)
			if err == bdog.ErrNotFound {
				basicError(w, http.StatusNotFound)
				return
			}
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
