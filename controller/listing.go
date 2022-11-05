package controller

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pbnjay/bdog"
)

func Listing(mod bdog.Model, table string) httprouter.Handle {
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
	}
}

// ListingFromSingle exposes a list of <table2> items related to a specified <table1> item.
// Aka a one-to-many relationship (e.g. /invoices/1234/products to list Multiple Products
// bought on a single Invoice)
func ListingFromSingle(mod bdog.Model, table1, table2 string) httprouter.Handle {
	drv, ok := mod.(bdog.Driver)
	if !ok {
		panic("Model does not implement Driver interface")
	}

	tab1 := mod.GetTable(table1)
	tab2 := mod.GetTable(table2)
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if r.Method != http.MethodGet {
			basicError(w, http.StatusMethodNotAllowed)
			return
		}

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

		mod.GetSubqueryMapping(tab1, tab2, key, opts)

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
	}
}
