package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

type Server struct {
	LetsEncrypt LetsEncryptConfig
	Routes      []Route
	Config
}

type Route struct {
	Handler  http.Handler
	Patterns []string
}

type Config struct {
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	ErrorLog          *log.Logger
}

type LetsEncryptConfig struct {
	AcceptTOS bool
	Email     string
	CachePath string
}

func (s *Server) ListenAndServeTLS(ctx context.Context, httpAddress, httpsAddress string) error {
	httpListener, err := getListener(httpAddress, ":http")
	if err != nil {
		return err
	}
	httpsListener, err := getListener(httpsAddress, ":https")
	if err != nil {
		return err
	}
	return s.ServeTLS(ctx, httpListener, httpsListener)
}

func (s *Server) ListenAndServe(ctx context.Context, httpAddress string) error {
	httpListener, err := getListener(httpAddress, ":http")
	if err != nil {
		return err
	}
	return s.Serve(ctx, httpListener)
}

func (s *Server) ServeTLS(ctx context.Context, httpListener, httpsListener net.Listener) error {
	if ctx == nil {
		ctx = context.Background()
	}
	handler, hostnames, err := s.getHandlerAndHostnames()
	if err != nil {
		return err
	}
	certManager := autocert.Manager{
		Prompt:     func(string) bool { return s.LetsEncrypt.AcceptTOS },
		Email:      s.LetsEncrypt.Email,
		Cache:      autocert.DirCache(s.LetsEncrypt.CachePath),
		HostPolicy: autocert.HostWhitelist(hostnames...),
	}
	errors := make(chan error)
	http := s.httpServer(certManager.HTTPHandler(nil), nil)
	https := s.httpServer(handler, certManager.TLSConfig())
	go func() { errors <- s.serve(ctx, http, httpListener) }()
	go func() { errors <- s.serve(ctx, https, httpsListener) }()
	err, _ = <-errors, <-errors
	return err
}

func (s *Server) Serve(ctx context.Context, httpListener net.Listener) error {
	if ctx == nil {
		ctx = context.Background()
	}
	handler, _, err := s.getHandlerAndHostnames()
	if err != nil {
		return err
	}
	http := s.httpServer(handler, nil)
	return s.serve(ctx, http, httpListener)
}

func (s *Server) getHandlerAndHostnames() (http.Handler, []string, error) {
	mux, hostnames := http.NewServeMux(), []string{}
	for _, route := range s.Routes {
		for _, pattern := range route.Patterns {
			mux.Handle(pattern, route.Handler)
			parts := strings.Split(pattern, "/")
			if len(parts) < 2 {
				return nil, nil, fmt.Errorf("pattern must be either {hostname}/... or /...: %s", pattern)
			}
			if hostname := parts[0]; hostname != "" {
				hostnames = append(hostnames, hostname)
			}
		}
	}
	return mux, hostnames, nil
}

func (s *Server) serve(ctx context.Context, httpServer *http.Server, l net.Listener) error {
	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)
	}()
	if httpServer.TLSConfig == nil {
		return httpServer.Serve(l)
	}
	httpServer.ListenAndServe()
	return httpServer.ServeTLS(l, "", "")
}

func (s *Server) httpServer(handler http.Handler, tls *tls.Config) *http.Server {
	return &http.Server{
		TLSConfig:         tls,
		Handler:           handler,
		ReadTimeout:       s.ReadTimeout,
		ReadHeaderTimeout: s.ReadHeaderTimeout,
		WriteTimeout:      s.WriteTimeout,
		IdleTimeout:       s.IdleTimeout,
		MaxHeaderBytes:    s.MaxHeaderBytes,
		ErrorLog:          s.ErrorLog,
	}
}

func getListener(address, fallbackAddress string) (net.Listener, error) {
	if address == "" {
		address = fallbackAddress
	}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return tcpKeepAliveListener{ln.(*net.TCPListener)}, nil
}

// TODO: copied from http/server.go - remove with https://github.com/golang/go/commit/1abf3aa55bb8b346bb1575ac8db5022f215df65a
type tcpKeepAliveListener struct{ *net.TCPListener }

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
