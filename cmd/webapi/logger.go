package main

import (
	"log"
	"net/http"
	"time"
)

type traceWriter struct {
	http.ResponseWriter
	statusCode int
	nbytes     int
}

func (w *traceWriter) WriteHeader(status int) {
	w.statusCode = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *traceWriter) Flush() {
	z := w.ResponseWriter
	if f, ok := z.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *traceWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = 200
	}
	w.nbytes = len(b)
	return w.ResponseWriter.Write(b)
}

func clf(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		trace := &traceWriter{ResponseWriter: w}

		handler.ServeHTTP(trace, r)
		log.Printf(`%s "%s %s %s" %d %d %s %dus`,
			r.RemoteAddr,
			r.Method, r.URL.Path,
			r.Proto,
			trace.statusCode, trace.nbytes,
			r.UserAgent(),
			time.Since(t)/time.Microsecond,
		)
	})
}
