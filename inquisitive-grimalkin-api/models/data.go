package models

import (
	"github.com/google/uuid"
	"time"
)

type Question struct {
	QuestionId uuid.UUID `json: "questionId,omitempty"`
	Asked      string    `json: "asked"`
	Asker      string    `json: "asker"`
	IsAnon     bool      `json: "isAnon"`
	Question   string    `json: "question"`
}

type QAndA struct {
	QuestionId uuid.UUID
	Asked      string
	Asker      string
	IsAnon     bool
	Question   string
	Answer     string
	AnsweredOn time.Time
}

type User struct {
}
