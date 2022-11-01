package controller

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pbnjay/bdog"
)

func Insert(mod bdog.Model, table string) httprouter.Handle {
	drv, ok := mod.(bdog.Driver)
	if !ok {
		panic("Model does not implement Driver interface")
	}

	tab := mod.GetTable(table)
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if r.Method != http.MethodPost {
			basicError(w, http.StatusMethodNotAllowed)
			return
		}

		opts := make(map[string][]string)
		r.ParseForm()
		for _, colname := range tab.Columns {
			vals, ok := r.Form[colname]
			if ok && len(vals) > 0 {
				opts[colname] = vals
			}
		}

		data, err := drv.Insert(tab, opts)
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
	}
}
