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

func (e *Errors) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (e *Errors) Wrap(next http.Handler) http.Handler {
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
	})
}
