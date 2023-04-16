package routers

import (
	"inquisitive-grimalkin/data"
	"net/http"
	"github.com/go-chi/chi/v5"
)

type QuestionsRouter struct {
	chi.Router
	likesRepo     data.CassandraQuestionsRepository
	questionsRepo data.CassandraLikesRepository
}

func NewQuestionsRouter() QuestionsRouter {

	r := chi.NewRouter()
	questionsRouter := QuestionsRouter{
		Router: r,
		likesRepo:     data.NewCassandraQuestionsRepository(),
		questionsRepo: data.NewCassandraLikesRepository(),
	}

	r.Get("/", questionsRouter.GetUnansweredQuestions())
	r.Post("/", questionsRouter.Ask())

	r.Post("/{question_id}/", questionsRouter.AnswerQuestion())
	r.Put("/{question_id}/", questionsRouter.UpdateAnswer())
	r.Delete("/{question_id}", questionsRouter.DeleteQAndA())

	r.Get("/{question_id}/likes", questionsRouter.GetLikesForQAndA())
	r.Put("/{question_id}/like", questionsRouter.LikeQAndA())
	r.Put("/{question_id}/unlike", questionsRouter.UnlikeQAndA())

	r.Post("/{question_id}/share/", questionsRouter.Share())
	r.Post("/{question_id}/share/twitter", questionsRouter.ShareToTwitter())

	return questionsRouter
}

func (r *QuestionsRouter) GetUnansweredQuestions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
	}
}

func (router *QuestionsRouter) Ask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
	}
}

func (r *QuestionsRouter) UpdateAnswer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (r *QuestionsRouter) DeleteQAndA() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (r *QuestionsRouter) AnswerQuestion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (r *QuestionsRouter) GetLikesForQAndA() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (r *QuestionsRouter) LikeQAndA() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (r *QuestionsRouter) UnlikeQAndA() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (r *QuestionsRouter) Share() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (r *QuestionsRouter) ShareToTwitter() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
