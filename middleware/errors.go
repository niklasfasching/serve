package middleware

import (
	"bytes"
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
	// rather than ServeHTTP and then inspecting i could move all the logic into the intercepting ResponseWriter!!
	// so WriteHeader would intercept if status == x
	// the responseWriter would have to carry around the mapping - not sure i want that - tradeoffs!
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w, errorBuffer: &bytes.Buffer{}}
		next.ServeHTTP(rw, r)
		if rw.status < 400 {
			return
		}
		path, ok := e.Mapping[rw.status]
		if !ok {
			w.WriteHeader(rw.status)
			w.Write(rw.errorBuffer.Bytes())
			return
		}
		f, err := os.Open(path)
		if err != nil {
			log.Printf("error opening %s: %s", path, err) // TODO: use ErrorLog
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(rw.status)
		_, err = io.Copy(w, f)
		if err != nil {
			log.Printf("could not copy error page to request %s: %s", path, err) // TODO: use ErrorLog
		}
	}), nil, nil
}

type errorResponseWriter struct {
	http.ResponseWriter
	done bool
}

func (r *errorResponseWriter) Write(bytes []byte) (count int, err error) {
	if !r.done {
		return r.ResponseWriter.Write(bytes)
	}
	return len(bytes), nil
}

func (r *errorResponseWriter) WriteHeader(status int) {
	if status < 400 {
		r.ResponseWriter.WriteHeader(status)
	}

	// lookup in errors map - otherwise
	done := true // lookup()

	if !done {
		r.ResponseWriter.WriteHeader(status)
	}
}
