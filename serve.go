package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/niklasfasching/serve/middleware"
	"github.com/niklasfasching/serve/server"
)

type Config struct {
	HTTPConfig struct {
		HTTPAddress  string
		HTTPSAddress string
		server.Config
	}
	LetsEncrypt  server.LetsEncryptConfig
	VirtualHosts []VirtualHost
}

type VirtualHost struct {
	Patterns    []string
	Middlewares []Middleware
}

type Handler interface {
	Wrap(http.Handler) http.Handler
	Start(context.Context) error
}

type Middleware struct {
	Name string
	Handler
}

var sortedHandlers = []Handler{
	&middleware.Static{},
	&middleware.Proxy{},
	&middleware.BasicAuth{},
	&middleware.Log{},
	&middleware.Errors{},
}

var handlers = map[string]Handler{}

func init() {
	for _, h := range sortedHandlers {
		name := strings.Split(fmt.Sprintf("%T", h), ".")[1]
		handlers[name] = h
	}
}

func Start(ctx context.Context, c *Config) error {
	wg, routes := NewWaitGroup(ctx), []server.Route{}
	for _, vhost := range c.VirtualHosts {
		sortedMiddlewares, err := sortMiddlewares(vhost.Middlewares)
		if err != nil {
			return err
		}
		handler := http.Handler(nil)
		for _, m := range sortedMiddlewares {
			wg.Start(m.Start)
			if handler = m.Wrap(handler); handler == nil {
				return fmt.Errorf("bad middleware for %v: %#v(%#v) -> nil", vhost.Patterns, m.Handler, handler)
			}
		}
		routes = append(routes, server.Route{Handler: handler, Patterns: vhost.Patterns})
	}

	httpConfig := c.HTTPConfig.Config
	if httpConfig.ErrorLog == nil {
		httpConfig.ErrorLog = log.New(os.Stderr, "", 0)
	}
	s := &server.Server{
		LetsEncrypt: c.LetsEncrypt,
		Routes:      routes,
		Config:      httpConfig,
	}
	if !c.LetsEncrypt.AcceptTOS {
		wg.Start(func(ctx context.Context) error {
			return s.ListenAndServe(ctx, c.HTTPConfig.HTTPAddress)
		})
	} else {
		wg.Start(func(ctx context.Context) error {
			return s.ListenAndServeTLS(ctx, c.HTTPConfig.HTTPAddress, c.HTTPConfig.HTTPSAddress)
		})
	}
	return wg.Wait()
}

func ReadConfig(path string) (*Config, error) {
	c := &Config{}
	return c, readJSON(path, c)
}

func ReadConfigDirectory(path string) (*Config, error) {
	c, err := ReadConfig(filepath.Join("serve.json"))
	if err != nil {
		return nil, err
	}
	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, fi := range fileInfos {
		if !strings.HasSuffix(fi.Name(), "_vhost.json") {
			continue
		}
		vh := VirtualHost{}
		if err := readJSON(filepath.Join(path, fi.Name()), &vh); err != nil {
			return nil, err
		}
		c.VirtualHosts = append(c.VirtualHosts, vh)
	}
	return c, nil
}

func (d *Middleware) UnmarshalJSON(bytes []byte) error {
	meta := &struct {
		Name    string
		Handler json.RawMessage
	}{}
	if err := json.Unmarshal(bytes, meta); err != nil {
		return err
	}
	if _, ok := handlers[meta.Name]; !ok {
		return fmt.Errorf("unknown middleware: %s", bytes)
	}
	v := reflect.New(reflect.TypeOf(handlers[meta.Name]).Elem()).Interface()
	if err := json.Unmarshal(meta.Handler, v); err != nil {
		return err
	}
	d.Name, d.Handler = meta.Name, v.(Handler)
	return nil
}
