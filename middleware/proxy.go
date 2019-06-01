package middleware

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	URL string
	url *url.URL
}

func (p *Proxy) Wrap(http.Handler) http.Handler {
	return httputil.NewSingleHostReverseProxy(p.url)
}

func (p *Proxy) Start(ctx context.Context) (err error) {
	if p.url, err = url.Parse(p.URL); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}
