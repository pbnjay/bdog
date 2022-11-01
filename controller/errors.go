package controller

import "net/http"

func basicError(w http.ResponseWriter, errCode int) {
	http.Error(w, http.StatusText(errCode), errCode)
}
