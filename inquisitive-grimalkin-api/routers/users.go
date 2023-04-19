package routers

import (
	"context"
	"encoding/json"
	"fmt"
	"inquisitive-grimalkin/data"
	"inquisitive-grimalkin/models"
	"inquisitive-grimalkin/services"
	"io"
	"log"
	"net/http"
	"time"
	"github.com/go-chi/chi/v5"
	jwt "github.com/golang-jwt/jwt"
)

type UsersRouter struct {
	chi.Router
	userRepository data.UsersRepository
	userService services.UsersService
}

func NewUsersRouter() UsersRouter {
	embeddableRouter := chi.NewRouter()
	r := UsersRouter{
		Router:         embeddableRouter,
		userRepository: data.NewCassandraUsersRepository(),
	}

	r.Post("/register", r.Register())
	r.Post("/login", r.Login())
	r.Post("/validate", r.Validate())
	//TODO: As a placeholder, we will be adding the follower to the path, but it should be noted that the follower username will be removed from the url and parsed from JWT
	r.Post("/follow/{followed}/follower/{follower}", r.Follow())
	//TODO: As a placeholder, we will be adding the follower to the path, but it should be noted that the follower username will be removed from the url and parsed from JWT
	r.Post("/unfollow/{followed}/follower/{follower}", r.Unfollow())
	r.Get("/{username}", r.SearchForUsername())

	return r
}


func (router *UsersRouter) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var userToBeRegisterd models.User
		userInBytes, err := io.ReadAll(r.Body)
		if err != nil {
			msg := fmt.Sprintf("failed to parse the request body to bytes %s", err)
			w.WriteHeader(http.StatusInternalServerError)	
			w.Write([]byte(msg))
			return
		}
		err = json.Unmarshal(userInBytes, &userToBeRegisterd)
		if err != nil {
			msg := fmt.Sprintf("failed to unmarshall the request body to user object %s", err)
			w.WriteHeader(http.StatusInternalServerError)	
			w.Write([]byte(msg))
			return
		}

		registeredUser, err := router.userRepository.Register(context.TODO(), userToBeRegisterd)
		if err != nil {
			msg := fmt.Sprintf("failed to create user %s", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(msg))
			return
		}

		registeredUserInString := fmt.Sprintf("%s", registeredUser)

		claimsForANormalUser := jwt.MapClaims{
			`nba`:(time.Now().Unix() + 31536000),
			`username`:registeredUser.Username,
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsForANormalUser)

		tokenString, err := token.SignedString([]byte(signingKey))
		if err != nil {
			msg := fmt.Sprintf("failed to sign the jwt %s", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(msg))
			return
		}
		w.Header().Set("Authorization", `bearer ` + tokenString)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(registeredUserInString))
	}
}

func (router *UsersRouter) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (router *UsersRouter) Validate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func (router *UsersRouter) Follow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		follower := chi.URLParam(r, "follower")
		following := chi.URLParam(r, "following")

		context := context.TODO()
		err := router.userService.Follow(context, follower, following)
		if err != nil {

			return
		}
	}	
}

func (router *UsersRouter) Unfollow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		follower := chi.URLParam(r, "follower")
		following := chi.URLParam(r, "following")

		context := context.TODO()
		err := router.userService.Unfollow(context, follower, following)
		if err != nil {
			return
		}
	}	
}

func (router *UsersRouter) SearchForUsername() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}	
}