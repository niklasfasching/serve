/*
We should not log access.log to journald

https://httpd.apache.org/docs/trunk/mod/mod_journald.html
Currently, systemd-journald is not designed for high-throughput logging and logging access_log to systemd-journald could decrease the performance a lot.
*/
package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"
)

var commonLogFormat = `{{ .remote }} - {{ .userAgent }} [{{ .timestamp }}] "{{ .method }} {{ .url }} {{ .proto }}" {{ .status }} {{ .size }}`

type Log struct {
	Path   string
	Format string

	accessLog *log.Logger
	format    func(interface{}) string
}

func (l *Log) Wrap(next http.Handler) (http.Handler, func(context.Context) error, error) {
	if l.Path == "" {
		return nil, nil, errors.New("log name must not be empty")
	}
	f, err := os.OpenFile(l.Path, os.O_RDWR|os.O_CREATE, 0664)
	if err != nil {
		return nil, nil, err
	}
	rf := &rotatingFile{path: l.Path, file: f}
	l.accessLog = log.New(rf, "", 0)
	l.format, err = newLogFormatter(l.Format)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw, timestamp := &logResponseWriter{ResponseWriter: w}, time.Now()
		next.ServeHTTP(w, r)
		l.accessLog.Print(l.format(map[string]interface{}{
			"remote":    maskIP(r.RemoteAddr),
			"userAgent": r.UserAgent(),
			"timestamp": timestamp,
			"proto":     r.Proto,
			"method":    r.Method,
			"url":       r.URL,
			"status":    rw.status,
			"size":      rw.count,
		}))
	}), rf.start, nil
}

type logResponseWriter struct {
	status int
	count  int
	http.ResponseWriter
}

func (r *logResponseWriter) Write(bytes []byte) (count int, err error) {
	count, err = r.ResponseWriter.Write(bytes)
	r.count += count
	return count, err
}

func (r *logResponseWriter) WriteHeader(status int) {
	r.ResponseWriter.WriteHeader(status)
	r.status = status
}
