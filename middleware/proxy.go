package middleware

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	URL string
}

func (p *Proxy) Wrap(http.Handler) (http.Handler, func(context.Context) error, error) {
	url, err := url.Parse(p.URL)
	if err != nil {
		return nil, nil, err
	}
	return httputil.NewSingleHostReverseProxy(url), nil, nil
}
