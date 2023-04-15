package models

import (
	"time"
	"github.com/google/uuid"
)

type Question struct {
	QuestionId uuid.UUID
	Asked string
	Asker string
	IsAnon bool
	Question string
}

type QAndA struct {
	QuestionId uuid.UUID
	Asked string
	Asker string
	IsAnon bool
	Question string
	Answer string
	AnsweredOn time.Time
}

type User struct {
	
}