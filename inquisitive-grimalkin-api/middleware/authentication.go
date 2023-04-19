package middleware

import (
	// "context"
	// "log"
	// "fmt"
	"inquisitive-grimalkin/utils"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"
)

var signingKey = ``

func init() {
	godotenv.Load()
	signingKey = os.Getenv("JWT_VERIFIER")
}

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

		authorizationHeader := r.Header.Get("Authorization")
		log.Println(authorizationHeader)
		authorizationMetadata := strings.Split(authorizationHeader, " ")
		log.Println(authorizationMetadata)
		// The first element of the array is the auth-scheme, can be either basic, bearer, etc.. (keep this so that it might be later used to provide authentication for diff. schemes)
		_ = authorizationMetadata[0]
		tokenString := authorizationMetadata[1]

		if tokenString == "" {
			unauthorizedHander := UnAuthorizedHandler{}
			unauthorizedHander.ServeHTTP(w, r)
			return
		}

		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(tokenString, claims, verifierWithKey)
		if err != nil {
			log.Printf("failed to parse jwt %s", err)
			return
		}
		username, ok := claims["username"].(string)
		if !ok {
			log.Printf("failed to retrieve username from token")
			return
		}

		context := utils.ContextWithUsername(r.Context(), username)
		r = r.WithContext(context)
		next.ServeHTTP(w, r)
	})
}

type UnAuthorizedHandler struct {}

func (h *UnAuthorizedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)  {
	w.WriteHeader(http.StatusUnauthorized)	
}


func verifierWithKey(t *jwt.Token) (interface{}, error) {
	return []byte(signingKey), nil	
}