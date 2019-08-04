package middleware

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
)

type Errors struct {
	Mapping map[int]string
}

func (e *Errors) Wrap(next http.Handler) (http.Handler, func(context.Context) error, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &errorResponseWriter{ResponseWriter: w, Mapping: e.Mapping}
		next.ServeHTTP(rw, r)
	}), nil, nil
}

type errorResponseWriter struct {
	http.ResponseWriter
	Mapping map[int]string
	done    bool
}

func (r *errorResponseWriter) Write(bytes []byte) (int, error) {
	if r.done {
		return len(bytes), nil
	}
	return r.ResponseWriter.Write(bytes)
}

func (r *errorResponseWriter) WriteHeader(status int) {
	if status < 400 {
		r.ResponseWriter.WriteHeader(status)
		return
	}
	path, ok := r.Mapping[status]
	if !ok {
		r.ResponseWriter.WriteHeader(status)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		log.Printf("error opening %s: %s", path, err)
		r.ResponseWriter.WriteHeader(status)
		return
	}
	defer f.Close()
	r.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	r.ResponseWriter.WriteHeader(status)
	if _, err := io.Copy(r, f); err != nil {
		log.Printf("could not copy error page to request %s: %s", path, err)
	}
	r.done = true
}
