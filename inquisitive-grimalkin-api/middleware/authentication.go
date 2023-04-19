package middleware

import (
	// "context"
	// "log"
	"net/http"
)

type pathsRequringNoAuthentication map[string]bool

func (m pathsRequringNoAuthentication) Has(path string) bool {
	return m[path]
}

var permissiblePathsWithNoAuthentication pathsRequringNoAuthentication= map[string]bool {
	"/users/register":true,
	"/users/login":true,
	"/users/logout":true,
	"/users/validate":true,
}


func JwtAuthenticationMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if permissiblePathsWithNoAuthentication.Has(path) {
			next.ServeHTTP(w, r)
			return
		}

		jwtToken := r.Header.Get("Authorization")
		if jwtToken == "" {
			unauthorizedHander := UnAuthorizedHandler{}
			unauthorizedHander.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type UnAuthorizedHandler struct {}

func (h *UnAuthorizedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)  {
	w.WriteHeader(http.StatusUnauthorized)	
}