package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/pbnjay/bdog"
)

func (c *Controller) Single(table string) {
	drv := c.mod.(bdog.Driver)
	tab := c.mod.GetTable(table)
	keypath := ":" + strings.Join(tab.Key, "/:")

	log.Printf("GET /%s/%s", table, keypath)
	apiGet := c.apiSpec.NewHandler("GET", "/"+table+"/"+keypath)
	apiGet.Summary = "Get one record from " + table

	rels := c.mod.ListRelatedTableNames(table)
	if len(rels) > 0 {
		relIncludes := []string{}

		for _, other := range rels {
			otherTab := c.mod.GetTable(other)
			colmaps := c.mod.GetRelatedTableMappings(table, other)

			for _, rights := range colmaps {
				for _, right := range rights {
					// if <right> is the PK for <other> then this is a to-one relationship, ok to nest:
					if otherTab.Key.IsEqual(right) {
						log.Printf("GET /%s/%s?include=%s", table, keypath, other)
						relIncludes = append(relIncludes, other)
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

	c.router.GET("/"+table+"/"+keypath, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
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
			opts["_nest"] = uq["include"]
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
