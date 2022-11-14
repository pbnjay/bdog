package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/pbnjay/bdog"
)

func (c *Controller) Delete(table string) {
	drv := c.mod.(bdog.Driver)
	tab := c.mod.GetTable(table)
	keypath := ":" + strings.Join(tab.Key, "/:")

	route := "/" + tab.PluralName(false) + "/" + keypath
	log.Println("DELETE", route)
	apiDelete := c.apiSpec.NewHandler("DELETE", route)
	apiDelete.Summary = "Delete a given " + tab.SingleName(true)
	c.router.DELETE(route, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if r.Method != http.MethodDelete {
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

		err := drv.Delete(tab, opts)
		if err != nil {
			log.Println(err)
			if err == bdog.ErrNotFound {
				basicError(w, http.StatusNotFound)
				return
			}
			basicError(w, http.StatusInternalServerError)
			return
		}

		err = json.NewEncoder(w).Encode(map[string]string{"message": "record successfully deleted"})
		if err != nil {
			log.Println(err)
			basicError(w, http.StatusInternalServerError)
			return
		}
	})
}
