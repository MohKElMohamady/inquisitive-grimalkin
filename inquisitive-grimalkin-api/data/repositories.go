package data

import (
	"context"
	"inquisitive-grimalkin/models"
	"github.com/google/uuid"
)

type QuestionsRepository interface {
	GetUnansweredQuestions(context.Context) ([]models.Question, error)
	Ask(context.Context, models.Question) (models.Question, error)
	UpdateAnswer(context.Context, models.Question) ([]models.Question, error)
	DeleteQandA(context.Context) (error)
	AnswerQuestion(context.Context, models.QAndA) (models.QAndA, error)
}

type LikesRepository interface {
	GetLikesForQAndA(context.Context, uuid.UUID) (int64, error)
	LikeQAndA(context.Context) error
	UnlikeQAndA(context.Context) error
}
