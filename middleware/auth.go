package middleware

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
)

type BasicAuth struct {
	User     string
	Password string
	Realm    string
}

func (ba *BasicAuth) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (ba *BasicAuth) Wrap(next http.Handler) http.Handler {
	eq := func(s1, s2 string) bool {
		return subtle.ConstantTimeCompare([]byte(s1), []byte(s2)) == 1
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || !eq(user, ba.User) || !eq(password, ba.Password) {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, ba.Realm))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
