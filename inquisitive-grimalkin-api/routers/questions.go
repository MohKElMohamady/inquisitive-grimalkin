package routers

import (
	"context"
	"encoding/json"
	"fmt"
	"inquisitive-grimalkin/data"
	"inquisitive-grimalkin/models"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type QuestionsRouter struct {
	chi.Router
	questionsRepository     data.CassandraQuestionsRepository
	likesRepository data.CassandraLikesRepository
}

func NewQuestionsRouter() QuestionsRouter {

	r := chi.NewRouter()
	questionsRouter := QuestionsRouter{
		Router: r,
		questionsRepository: data.NewCassandraQuestionsRepository(),
		likesRepository: data.NewCassandraLikesRepository(),
	}

	r.Get("/", questionsRouter.GetUnansweredQuestions())
	r.Post("/", questionsRouter.Ask())

	r.Post("/{question_id}/answer/", questionsRouter.AnswerQuestion())
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
		defer r.Body.Close()

		reqInBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to parsed the request"))
			return
		}
		
		question := models.Question{}
		err = json.Unmarshal(reqInBytes, &question)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to convert the request from json to bytes"))
			return
		}

		context := context.TODO()
		q, err := router.questionsRepository.Ask(context, question)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		resInBytes, err := json.Marshal(q)	
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		log.Println("Successfully handled the request of asking a question")	
		w.WriteHeader(http.StatusCreated)
		w.Write(resInBytes)
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

func (router *QuestionsRouter) AnswerQuestion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}

func (router *QuestionsRouter) GetLikesForQAndA() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questionId := chi.URLParam(r, "question_id")	
		questionUuid := uuid.MustParse(questionId)
		likes, err := router.likesRepository.GetLikesForQAndA(context.TODO(), questionUuid)	
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("failed to fetch likes for answer with id %s %s", questionId, err)))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"likes":%v}`, likes)))
	}
}

func (router *QuestionsRouter) LikeQAndA() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questionId := chi.URLParam(r, "question_id")	
		questionUuid := uuid.MustParse(questionId)
		err := router.likesRepository.LikeQAndA(context.TODO(), questionUuid)	
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("failed to like answer with id %s %s", questionId, err)))
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (router *QuestionsRouter) UnlikeQAndA() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questionId := chi.URLParam(r, "question_id")	
		questionUuid := uuid.MustParse(questionId)
		err := router.likesRepository.UnlikeQAndA(context.TODO(), questionUuid)	
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("failed to like answer with id %s %s", questionId, err)))
			return
		}
		w.WriteHeader(http.StatusOK)
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
