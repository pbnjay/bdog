package controller

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pbnjay/bdog"
)

func Single(mod bdog.Model, table string) httprouter.Handle {
	drv, ok := mod.(bdog.Driver)
	if !ok {
		panic("Model does not implement Driver interface")
	}

	tab := mod.GetTable(table)
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if r.Method != http.MethodGet {
			basicError(w, http.StatusMethodNotAllowed)
			return
		}

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
	}
}
