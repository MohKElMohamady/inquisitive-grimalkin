package data

import (
	"context"
	"github.com/google/uuid"
	"inquisitive-grimalkin/models"
)

type QuestionsRepository interface {
	GetUnansweredQuestionsForUser(context.Context, string) ([]models.Question, error)
	Ask(context.Context, models.Question) (models.Question, error)
	AnswerQuestion(context context.Context,questionId uuid.UUID , qAndA models.QAndA) (models.QAndA, error)
	UpdateAnswer(context.Context, models.QAndA) (models.QAndA, error)
	DeleteQAndA(context.Context, models.QAndA) error
	PostAnswerToFollowersHomefeed(context.Context, models.QAndA,...models.User) (error)
	UpdateAnswerToFollowersHomefeed(context.Context, models.QAndA, ...models.User) (error)
	DeleteAnswerFromFollowersHomefeed(context.Context, models.QAndA, ...models.User) (error)
}

type LikesRepository interface {
	GetLikesForQAndA(context.Context, uuid.UUID) (int64, error)
	CreateLikesEntryForQAndA(context.Context, uuid.UUID) (int64, error)
	LikeQAndA(context.Context, uuid.UUID) error
	UnlikeQAndA(context.Context, uuid.UUID) error
	DeleteQAndA(context.Context, uuid.UUID) error
}

type UsersRepository interface {
	DoesUserExist(context.Context, models.User) (bool, error)
	Register(context.Context, models.User) (models.User, error)
	Login(context.Context, models.User) (models.User, error)
	Delete(context.Context, models.User) error
	UpdateLoginDetails(context.Context, models.User) (models.User, error)
	Follow(context context.Context, follower models.User, followed models.User) (error)
	Unfollow(context context.Context, follower models.User, followed models.User) (error)
	FindFollowersOfUser(context context.Context, username string) ([]models.User, error)
	SearchForUsername(context.Context, string) ([]models.User, error)
}