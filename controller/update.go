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

func (c *Controller) Update(table string) {
	tab := c.mod.GetTable(table)
	drv := tab.Driver
	keypath := ":" + strings.Join(tab.Key, "/:")

	route := "/" + tab.PluralName(false) + "/" + keypath
	log.Println("PATCH", route)
	apiPatch := c.apiSpec.NewHandler("PATCH", route)
	apiPatch.Summary = "Update (part of) " + tab.SingleName(true) + " details"

	c.router.PATCH(route, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if r.Method != http.MethodPut {
			basicError(w, http.StatusMethodNotAllowed)
			return
		}
		if c.CORSEnabled {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Content-Type", "application/json")

		opts := make(map[string][]string)
		if r.Header.Get("Content-Type") == "application/json" {
			data := make(map[string]interface{})
			err := json.NewDecoder(r.Body).Decode(&data)
			if err != nil {
				log.Println(err)
				basicError(w, http.StatusBadRequest)
				return
			}
			for _, colname := range tab.Columns {
				val, ok := data[colname]
				if ok {
					opts[colname] = append(opts[colname], fmt.Sprint(val))
				}
			}
		} else {
			r.ParseForm()
			for _, colname := range tab.Columns {
				vals, ok := r.Form[colname]
				if ok && len(vals) > 0 {
					opts[colname] = vals
				}
			}
		}

		for _, colname := range tab.Key {
			key := params.ByName(colname)
			delete(opts, colname)
			opts[colname] = append(opts[colname], key)
		}

		data, err := drv.Update(tab, opts)
		if err != nil {
			log.Println(err)
			if err == bdog.ErrNotFound {
				basicError(w, http.StatusNotFound)
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
