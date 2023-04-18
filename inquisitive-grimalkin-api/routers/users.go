package routers

import (
	"context"
	"encoding/json"
	"fmt"
	"inquisitive-grimalkin/data"
	"inquisitive-grimalkin/models"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type UsersRouter struct {
	chi.Router
	userRepository data.UsersRepository
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
		w.WriteHeader(http.StatusOK)
		//TODO Create a token and return it instead of returning the user
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

func (router *UsersRouter) SearchForUsername() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}	
}