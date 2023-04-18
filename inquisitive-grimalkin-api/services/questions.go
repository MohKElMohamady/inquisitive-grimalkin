package services

import (
	"context"
	"fmt"
	"inquisitive-grimalkin/data"
	"inquisitive-grimalkin/models"
	"github.com/google/uuid"
	// "sync"
)

type QuestionsService struct {
	questionsRepository data.CassandraQuestionsRepository
	usersRepository data.CassandraUsersRepository
	likesRepository data.CassandraLikesRepository
}

func (s *QuestionsService) Ask(context context.Context, q models.Question) (models.Question, error) {
	q, err := s.questionsRepository.Ask(context, q)	
	return q, err
}

 /*
  - Answering the question will have multiple steps:
  - 1) Delete the question from the original the questions_by_user table.
  - 2) Add the answered question in the Q&A question i.e. q_and_a_user
  - 3) Find the followers of that user and post it to their timelines i.e. search for all the followers of the asked person and post save in their
    q_and_a_follower table
  - Initial idea: Retrieve all the followers and the calculate their length i.e. number of followers, which a touch of concurreny, spawn number of goroutines
    equal to the number of followers and then write to their timeline. (this might be a huge overhead if the user has large number of followers). Maybe
    check the number of followers and try to rationalize i.e. distribute the actions of saving in the table depending on how big the followers are?
  - 4) Since the counter tables do not allow insertion, we will just update the q&a of that like to be added and set to zero.
*/
func (s *QuestionsService) AnswerQuestion(context context.Context,questionUuidInString string, qAndA models.QAndA) (models.QAndA, error) {

	// Updating the question with the answer and posting the answer to the followers timeline can be done in parallel
	// wg := sync.WaitGroup{}
	// wg.Add(2)

	// Step 1 Answer the question and delete the question from the table and insert it to the q&a table	
	parsedQuestionUuid, err := uuid.Parse(questionUuidInString)
	if err != nil {
		return models.QAndA{}, fmt.Errorf("failed to post answer to question with id %s %s", parsedQuestionUuid, err)
	}
	answeredQuestion, err := s.questionsRepository.AnswerQuestion(context, parsedQuestionUuid, qAndA)
	if err != nil {
		return models.QAndA{}, err
	}
	// Step 2 Fetching the followers of the user so that we can add to their homefeeds 
	followers, err := s.usersRepository.FindFollowersOfUser(context, answeredQuestion.Asked)
	if err != nil {
		return models.QAndA{}, err 
	}

	// Step 3 Post the answer to the followers fields
	err = s.questionsRepository.PostAnswerToFollowersHomefeed(context, answeredQuestion, followers)
	if err != nil {
		return models.QAndA{}, err
	}

	// Step 4 Insert the answer to the likes counter table
	_, err = s.likesRepository.CreateLikesEntryForQAndA(context, parsedQuestionUuid)
	if err != nil {
		return models.QAndA{}, err
	}

	return qAndA, nil
}


func (q *QuestionsService) UpdateAnswer(context context.Context, qAndA models.QAndA) (models.QAndA, error) {


	// TODO: Step 3 post all the answer to the asked's followers
	// followers := q.usersRepository.FindFollowersForUsers(context, qAndA)
	// q.questionsRepository.UpdateFollowersTimeline(context, qAndA, followers)


	// TODO : Step 4 Add the Q&A to the likes counter table
	updatedAnswer,err := q.questionsRepository.UpdateAnswer(context, qAndA)
	if err != nil {
		return updatedAnswer, err
	}



	return qAndA, nil
}


