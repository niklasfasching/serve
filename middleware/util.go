package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

type rotatingFile struct {
	path   string
	file   *os.File
	buffer bytes.Buffer
	locked bool
}

type logResponseWriter struct {
	status int
	count  int
	http.ResponseWriter
}

var ipv4Mask = net.CIDRMask(16, 32)  // 255.255.0.0
var ipv6Mask = net.CIDRMask(56, 128) // ffff:ffff:ffff:ff00::

func (l *logResponseWriter) Write(bytes []byte) (int, error) {
	count, err := l.ResponseWriter.Write(bytes)
	l.count += count
	return count, err
}

func (l *logResponseWriter) WriteHeader(status int) {
	l.status = status
	l.ResponseWriter.WriteHeader(status)
}

func (rf *rotatingFile) Write(bytes []byte) (int, error) {
	if rf.locked {
		return rf.buffer.Write(bytes)
	}
	return rf.file.Write(bytes)
}

func (rf *rotatingFile) rotate() error {
	rf.locked = true
	if err := rf.file.Close(); err != nil {
		return err
	}
	backupPath := fmt.Sprintf("%s.%s", rf.path, time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
	if err := os.Rename(rf.path, backupPath); err != nil {
		return err
	}
	// TODO: cleanup old log files, gzip
	f, err := os.OpenFile(rf.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	rf.file = f
	rf.locked = false
	_, err = rf.file.Write(rf.buffer.Bytes())
	rf.buffer = bytes.Buffer{}
	return err
}

func (rf *rotatingFile) start(ctx context.Context) error {
	for {
		tmrw := time.Now().AddDate(0, 0, 1)
		midnight := time.Date(tmrw.Year(), tmrw.Month(), tmrw.Day(), 0, 0, 0, 0, tmrw.Location())
		select {
		case <-time.After(midnight.Sub(time.Now())):
			if err := rf.rotate(); err != nil {
				return fmt.Errorf("could not rotate %s: %s", rf.path, err)
			}
		case <-ctx.Done():
			return rf.file.Close()
		}
	}
}

func newLogFormatter(format string) (func(interface{}) string, error) {
	if format == "" {
		format = commonLogFormat
	}
	logTemplate, err := template.New("logFormat").Parse(format)
	return func(data interface{}) string {
		s := &strings.Builder{}
		if err := logTemplate.Execute(s, data); err != nil {
			panic(err)
		}
		return s.String()
	}, err
}

func maskIP(remoteAddress string) string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		return "-"
	}
	ip := net.ParseIP(host)
	if ip.To4() != nil {
		return ip.Mask(ipv4Mask).String()
	}
	return ip.Mask(ipv6Mask).String()
}
