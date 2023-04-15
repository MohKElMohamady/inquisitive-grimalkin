package data

import (
	//"context"
	"context"
	"crypto/tls"
	"fmt"
	"inquisitive-grimalkin/models"
	"inquisitive-grimalkin/utils"
	"log"
	"os"
	"sync"
	"time"

	//"github.com/google/uuid"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stargate/stargate-grpc-go-client/stargate/pkg/auth"
	"github.com/stargate/stargate-grpc-go-client/stargate/pkg/client"
	"github.com/stargate/stargate-grpc-go-client/stargate/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var cassandraRemoteUri string
var cassandraClientId string
var cassandraClientSecret string
var cassandraBearerToken string

var questionsByUserTableDDL = `CREATE TABLE IF NOT EXISTS main.questions_by_user (asked text, question_id timeuuid, asker text, is_anon boolean, question text,
								PRIMARY KEY ((asked), question_id));`
var qAndAByUserTableDDL = `CREATE TABLE IF NOT EXISTS main.q_and_a_users (asked text, question_id timeuuid, asker text, is_anon boolean, question text, answer text,
								PRIMARY KEY ((asked), question_id));`

/*
  - A problem that needs to be solved is showing the answered questions of the users that one person follows.
  - The implementation is inspired by what was mentioned in Martin Kleppmann's Designing Data-Intensive Application (P11-13) about Twitter handling the display of
  - home timelines for users:
    EXCERPT STARTS
  - let’s consider Twitter as an example, using data published in November 2012 [16]. Two of Twitter’s main operations are:
  - Post tweet: A user can publish a new message to their followers (4.6k requests/sec on aver‐ age, over 12k requests/sec at peak).
  - Home timeline :A user can view tweets posted by the people they follow (300k requests/sec).
  - Simply handling 12,000 writes per second (the peak rate for posting tweets) would be fairly easy. However, Twitter’s scaling challenge is not primarily
  - due to tweet volume, but due to fan-outii—each user follows many people, and each user is followed by many people.
  - There are broadly two ways of implementing these two operations:
  - 1) Posting a tweet simply inserts the new tweet into a global collection of tweets. When a user requests their home timeline, look up all the people
  - they follow, find all the tweets for each of those users, and merge them (sorted by time). In a relational database you could write a query such as:
  - SELECT tweets.*, users.* FROM tweets
  - JOIN users ON tweets.sender_id = users.id
  - JOIN follows ON follows.followee_id = users.id
  - WHERE follows.follower_id = current_user;
  - 2) Maintain a cache for each user’s home timeline—like a mailbox of tweets for each recipient user.
  - When a user posts a tweet, look up all the people who follow that user, and insert the new tweet into each of their home timeline caches.
  - The request to read the home timeline is then cheap, because its result has been computed ahead of time.
  - The first version of Twitter used approach 1, but the systems struggled to keep up with the load of home timeline queries,
  - so the company switched to approach 2.
  - This works better because the average rate of published tweets is almost two orders of magnitude lower than the rate of home timeline reads, and so in
  - this case it’s prefera‐ ble to do more work at write time and less at read time.
    EXCERPT ENDS
  - Once the question is answered by the asked, we will look for are the people who follow the person who answered the question and post to the table named
  - q_and_a_by_follower where the partition key is the followers of the person who answered the question and the clustering column is the question time uuid
  - which allow us to arrange it chronologically.
*/
var qAndAByFollowerDDL = `CREATE TABLE IF NOT EXISTS main.q_and_a_followers (follower text, asked text, question_id timeuuid, asker text, is_anon boolean, question text, answer text, 
							PRIMARY KEY ((follower), question_id));`
var qAndALikesDDL = `CREATE TABLE IF NOT EXISTS main.q_and_a_likes (question_id timeuuid, likes counter, PRIMARY KEY ((question_id)));`

var cassandraConnectionClient = sync.Pool{
	New: func() any {
		config := &tls.Config{InsecureSkipVerify: false}
		conn, err := grpc.Dial(
			cassandraRemoteUri,
			grpc.WithTransportCredentials(credentials.NewTLS(config)),
			grpc.WithBlock(),
			grpc.FailOnNonTempDialError(true),
			grpc.WithPerRPCCredentials(auth.NewStaticTokenProvider(cassandraBearerToken)),
		)
		if err != nil {
			log.Fatalf("failed to connect to remote Cassandra instance from datastax %s\n", err)
		}

		stargateClient, err := client.NewStargateClientWithConn(conn)
		if err != nil {
			log.Fatalf("failed to create stargate client %s\n", err)
		}
		return stargateClient
	},
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("something went wrong in loading the variables %s\n", err)
	}
	cassandraRemoteUri = os.Getenv("CASSANDRA_REMOTE_URI")
	cassandraClientId = os.Getenv("CASSANDRA_CLIENT_ID")
	cassandraClientSecret = os.Getenv("CASSANDRA_CLIENT_SECRET")
	cassandraBearerToken = os.Getenv("CASSANDRA_BEARER_TOKEN")

	tableCreationSynchronizer := sync.WaitGroup{}
	tableCreationSynchronizer.Add(2)

	go func() {
		cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
		_, err := cassandraClient.ExecuteQuery(&proto.Query{Cql: questionsByUserTableDDL})
		if err != nil {
			log.Fatalf("failed to create questions_by_user table %s\n", err)
		}

		_, err = cassandraClient.ExecuteQuery(&proto.Query{Cql: qAndAByUserTableDDL})
		if err != nil {
			log.Fatalf("failed to create q_and_a_user table %s\n", err)
		}
		tableCreationSynchronizer.Done()
	}()

	go func() {
		cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
		defer cassandraConnectionClient.Put(cassandraClient)
		_, err := cassandraClient.ExecuteQuery(&proto.Query{Cql: qAndAByFollowerDDL})
		if err != nil {
			log.Fatalf("failed to create questions_by_user table %s\n", err)
		}

		_, err = cassandraClient.ExecuteQuery(&proto.Query{Cql: qAndALikesDDL})
		if err != nil {
			log.Fatalf("failed to create q_and_a_user table %s\n", err)
		}
		tableCreationSynchronizer.Done()
	}()

	tableCreationSynchronizer.Wait()
	log.Printf("successfully created all tables ")
}

func Foo() {
	log.Printf("...")
}

type CassandraQuestionsRepository struct {
}

func (c *CassandraQuestionsRepository) GetUnansweredQuestions(_ context.Context) ([]models.Question, error) {
	panic("not implemented") // TODO: Implement
}

func (c *CassandraQuestionsRepository) Ask(ctx context.Context, q models.Question) (models.Question, error) {
	panic("not implemented") // TODO: Implement
}

func (c *CassandraQuestionsRepository) UpdateAnswer(_ context.Context, _ models.Question) ([]models.Question, error) {
	panic("not implemented") // TODO: Implement
}

func (c *CassandraQuestionsRepository) DeleteQandA(_ context.Context) error {
	panic("not implemented") // TODO: Implement
}

func (c *CassandraQuestionsRepository) AnswerQuestion(_ context.Context, _ models.QAndA) (models.QAndA, error) {
	panic("not implemented") // TODO: Implement
}

type CassandraLikesRepository struct {
}

func (c *CassandraLikesRepository) GetLikesForQAndA(ctx context.Context, qAndAId uuid.UUID) (int64, error) {
	panic("not implemented") // TODO: Implement
}

func (c *CassandraLikesRepository) LikeQAndA(_ context.Context, qAndAUuid uuid.UUID) error {
	panic("not implemented") // TODO: Implement
}

func (c *CassandraLikesRepository) UnlikeQAndA(_ context.Context, qAndAUuid uuid.UUID) error {
	panic("not implemented") // TODO: Implement
}
