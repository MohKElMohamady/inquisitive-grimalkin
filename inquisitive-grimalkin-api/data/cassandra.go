package data

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stargate/stargate-grpc-go-client/stargate/pkg/auth"
	"github.com/stargate/stargate-grpc-go-client/stargate/pkg/client"
	"github.com/stargate/stargate-grpc-go-client/stargate/pkg/proto"
	"inquisitive-grimalkin/models"
	"inquisitive-grimalkin/utils"
	"log"
	"os"
	"sync"
	"time"
	// "golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type LikeAction int

const (
	Like LikeAction = iota
	Dislike
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
  - The implementation is inspired by what was mentioned in Martin Kleppmann's Designing Data-Intensive Application (P11-13) about Twitter handling the kkjkjdisplay of
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

var usersDDL = `CREATE TABLE IF NOT EXISTS main.users (username text, email text, first_name text, last_name text, password text, created_on timeuuid, PRIMARY KEY ((username)));`

/*
 * This table will have the user as the partition key and the followers i.e. other users as clustering keys
 */
var userByFollowersDDL = `CREATE TABLE IF NOT EXISTS main.followers_by_user (followed text, follower text, PRIMARY KEY ((followed), follower));`

/*
 * Instead of using grouping by and counting the number of those who the user follows and who follow him, we will create two counter tables each for following 
   and followers
 */
var followersOfUserCounterDDL = `CREATE TABLE IF NOT EXISTS main.followers_of_user_counter (username text, followers counter, PRIMARY KEY ((username)));`
var followingByUserCounterDDL = `CREATE TABLE IF NOT EXISTS main.followed_by_user_counter (username text, followering counter, PRIMARY KEY ((username)));`

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
	tableCreationSynchronizer.Add(4)

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

	go func() {
		cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
		defer cassandraConnectionClient.Put(cassandraClient)
		_, err := cassandraClient.ExecuteQuery(&proto.Query{Cql: usersDDL})
		if err != nil {
			log.Fatalf("failed to create users table %s\n", err)
		}

		_, err = cassandraClient.ExecuteQuery(&proto.Query{Cql: userByFollowersDDL})
		if err != nil {
			log.Fatalf("failed to create users_by_followers table %s\n", err)
		}
		tableCreationSynchronizer.Done()

	}()

	go func() {
		cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
		defer cassandraConnectionClient.Put(cassandraClient)
		_, err := cassandraClient.ExecuteQuery(&proto.Query{Cql: followersOfUserCounterDDL})
		if err != nil {
			log.Fatalf("failed to create followers by user counter table %s\n", err)
		}

		_, err = cassandraClient.ExecuteQuery(&proto.Query{Cql: followingByUserCounterDDL})
		if err != nil {
			log.Fatalf("failed to create following by user counter table %s\n", err)
		}
		tableCreationSynchronizer.Done()

	}()
	tableCreationSynchronizer.Wait()
	log.Printf("successfully created all tables")
}

func NewCassandraQuestionsRepository() CassandraQuestionsRepository {
	return CassandraQuestionsRepository{}
}

type CassandraQuestionsRepository struct {
}

func (c *CassandraQuestionsRepository) GetUnansweredQuestionsForUser(context context.Context, askedUser string) ([]models.Question, error) {
	cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
	defer cassandraConnectionClient.Put(cassandraClient)

	if askedUser == "" {
		return nil, fmt.Errorf("cannot fetch the unanswered questions for no one")
	}

	getUnAnsweredQuestionForUserQuery := &proto.Query{
		Cql: "SELECT * FROM main.questions_by_user WHERE asked = ?;",
		Values: &proto.Values{
			Values: []*proto.Value{
				{Inner: &proto.Value_String_{askedUser}},
			},
		},
	}

	res, err := cassandraClient.ExecuteQuery(getUnAnsweredQuestionForUserQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unanswered questions for %s %s", askedUser, err)
	}

	unansweredQuestions := []models.Question{}
	for _, row := range res.GetResultSet().Rows {
		parsedQuestionUuid, err := cassandraUuidToGoogleUuid(row.Values[1])
		if err != nil {
			log.Printf("failed to parse the uuid of one question %s\n ", err)
		}
		q := models.Question{
			QuestionId: parsedQuestionUuid,
			Asked:      row.Values[0].GetString_(),
			Asker:      row.Values[2].GetString_(),
			IsAnon:     row.Values[3].GetBoolean(),
			Question:   row.Values[4].GetString_(),
		}
		unansweredQuestions = append(unansweredQuestions, q)
	}

	return unansweredQuestions, nil
}

func (c *CassandraQuestionsRepository) Ask(ctx context.Context, q models.Question) (models.Question, error) {
	cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
	defer cassandraConnectionClient.Put(cassandraClient)

	err := utils.ValidateQuestion(q)
	if err != nil {
		return models.Question{}, fmt.Errorf("failed to validate the question %s", err)
	}

	generatedQuestionId, err := uuid.NewUUID()
	if err != nil {
		return models.Question{}, fmt.Errorf("failed to ask this question %s", err)
	}

	cassandraCompliantUuid, err := googleUuidToCassandraUuid(generatedQuestionId)
	if err != nil {
		return models.Question{}, fmt.Errorf("failed to ask this question %s, try again later", err)
	}

	AskQuery := &proto.Query{Cql: `INSERT INTO main.questions_by_user (asked , question_id , asker , is_anon , question) 
									VALUES (?, ?, ?, ?, ?);`,
		Values: &proto.Values{
			Values: []*proto.Value{
				{Inner: &proto.Value_String_{q.Asked}},
				{Inner: &proto.Value_Uuid{cassandraCompliantUuid}},
				{Inner: &proto.Value_String_{q.Asker}},
				{Inner: &proto.Value_Boolean{q.IsAnon}},
				{Inner: &proto.Value_String_{q.Question}},
			},
		},
	}

	/*
	 * Resposne is deliberatly ignored because as per Cassandra Standard, inserted values do not get returned as part of response just like SQL
	 */
	_, err = cassandraClient.ExecuteQuery(AskQuery)
	if err != nil {
		return models.Question{}, err
	}

	persistedQuestion := models.Question{
		QuestionId: generatedQuestionId,
		Asked:      q.Asked,
		Asker:      q.Asker,
		IsAnon:     q.IsAnon,
		Question:   q.Question,
	}
	log.Printf("successfully persisted the question %s at %s\n", persistedQuestion, time.Now())

	return persistedQuestion, nil
}

func (c *CassandraQuestionsRepository) AnswerQuestion(ctx context.Context,questionId uuid.UUID, qAndA models.QAndA) (models.QAndA, error) {
	cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
	defer cassandraConnectionClient.Put(cassandraClient)

	cassandraCompliantQAndAUuid, err := googleUuidToCassandraUuid(questionId)
	if err != nil {
		return models.QAndA{}, fmt.Errorf("failed to answer the question and parsing the uuid %s", err)
	}

	// TODO: Refactor the deletion into a separate table
	deleteTheQuestionWithoutAnswerQuery := `delete FROM main.questions_by_user WHERE asked = ? AND question_id = ?;`
	_, err = cassandraClient.ExecuteQuery(&proto.Query{
		Cql: deleteTheQuestionWithoutAnswerQuery,
		Values: &proto.Values{
			Values: []*proto.Value{
				&proto.Value{Inner: &proto.Value_String_{qAndA.Asked}},
				&proto.Value{Inner: &proto.Value_Uuid{cassandraCompliantQAndAUuid}},
			},
		},
	})
	if err != nil {
		return models.QAndA{}, fmt.Errorf("failed to answer the question, unable to delete the question before reposting %s", err)
	}

	qAndAUuid, err := uuid.NewUUID()
	if err != nil {
		return models.QAndA{}, fmt.Errorf("failed to generate new uuid for answer to question%s", err)
	}

	cassandraCompliantQAndAUuid, err = googleUuidToCassandraUuid(qAndAUuid)
	if err != nil {
		return models.QAndA{}, fmt.Errorf("failed to parse a cassandra compliant uuid for the answer %s", err)
	}

	// Step 2 insert the q&a into the table that will appear to the asked person
	insertAnsweredQuestionQuery := `insert INTO main.q_and_a_users 
									(asked , question_id , answer , asker , is_anon , question ) 
									VALUES (?, ? ,?, ?, ?, ?);`
	_, err = cassandraClient.ExecuteQuery(&proto.Query{
		Cql: insertAnsweredQuestionQuery,
		Values: &proto.Values{
			Values: []*proto.Value{
				{Inner: &proto.Value_String_{qAndA.Asked}},
				{Inner: &proto.Value_Uuid{cassandraCompliantQAndAUuid}},
				{Inner: &proto.Value_String_{qAndA.Answer}},
				{Inner: &proto.Value_String_{qAndA.Asker}},
				{Inner: &proto.Value_Boolean{qAndA.IsAnon}},
				{Inner: &proto.Value_String_{qAndA.Question}},
			},
		},
	})
	if err != nil {
		return models.QAndA{}, fmt.Errorf("failed to save the answer to the question %s", err)
	}

	qAndA.QuestionId = qAndAUuid
	return qAndA, nil
}

func (c *CassandraQuestionsRepository) UpdateAnswer(ctx context.Context, qAndA models.QAndA) (models.QAndA, error) {
	panic("not implemented") // TODO: implement
}

func (c *CassandraQuestionsRepository) DeleteQAndA(context context.Context, qAndA models.QAndA) error {
	panic("not implemented") // TODO: implement
}

func NewCassandraLikesRepository() CassandraLikesRepository {
	return CassandraLikesRepository{}
}

type CassandraLikesRepository struct {
}

/*
 * Later on when we fetch the answered questions of people who one user follows, we need to fetch the likes for each question, now this might be a little
 * bit slow if we do it sequentially, but since we are going to use pagination with constant number of questions per "load more", this constant number will
 * help us accelerate things a bit, where we can spawn a goroutine for each number of question that is loaded and we read the number of likes.
 * This will substantially speed up the fetching of all the likes
 */
func (c *CassandraLikesRepository) GetLikesForQAndA(ctx context.Context, qAndAId uuid.UUID) (int64, error) {
	cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
	defer cassandraConnectionClient.Put(cassandraClient)
	cassandraCompliantUuid, err := googleUuidToCassandraUuid(qAndAId)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch the likes for a specific Q&A with id %s", qAndAId)
	}

	fetchLikesForQAndAQuery := &proto.Query{
		Cql: `SELECT * FROM main.q_and_a_likes WHERE question_id = ?`,
		Values: &proto.Values{
			Values: []*proto.Value{
				&proto.Value{Inner: &proto.Value_Uuid{cassandraCompliantUuid}},
			},
		},
	}
	res, err := cassandraClient.ExecuteQueryWithContext(fetchLikesForQAndAQuery, ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch likes for the q&a %s", err)
	}

	var likes int64 = 0
	log.Printf("the total size of the result set is %v", len(res.GetResultSet().Rows))
	for _, v := range res.GetResultSet().Rows {
		_, err := cassandraUuidToGoogleUuid(v.Values[0])
		likes = v.Values[1].GetInt()
		if err != nil {
			log.Fatalln(err)
		}
	}

	return likes, nil
}

func (c *CassandraLikesRepository) CreateLikesEntryForQAndA(context context.Context, qAndAId uuid.UUID) (int64, error) {

	cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
	defer cassandraConnectionClient.Put(cassandraClient)
	cassandraCompliantQAndAUuid, err := googleUuidToCassandraUuid(qAndAId)
	if err != nil {
		return 0, fmt.Errorf("failed to insert the Q&A to the likes counter table %s", err)
	}

	addQAndAToLikesQuery := `update main.q_and_a_likes SET likes = likes + 0 WHERE question_id = ?;`
	_, err = cassandraClient.ExecuteQuery(&proto.Query{
		Cql: addQAndAToLikesQuery,
		Values: &proto.Values{
			Values: []*proto.Value{
				{Inner: &proto.Value_Uuid{cassandraCompliantQAndAUuid}},
			},
		},
	})
	if err != nil {
		return -1, fmt.Errorf("failed to create a likes counter for question with id %s %s", qAndAId, err)
	}
	return 0, nil

}

func (c *CassandraLikesRepository) LikeQAndA(ctx context.Context, qAndAUuid uuid.UUID) error {
	cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
	defer cassandraConnectionClient.Put(cassandraClient)

	q, err := updateLikesQuery(Like, qAndAUuid)
	if err != nil {
		return fmt.Errorf("failed to like the q&a %s", err)
	}

	/*
	 * Resposne is deliberatly ignored because as per Cassandra Standard, inserted values do not get returned as part of response just like SQL
	 */
	_, err = cassandraClient.ExecuteQuery(q)
	if err != nil {
		return fmt.Errorf("failed to like the q&a %s\n", err)
	}

	return nil
}

func (c *CassandraLikesRepository) UnlikeQAndA(ctx context.Context, qAndAUuid uuid.UUID) error {
	cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
	defer cassandraConnectionClient.Put(cassandraClient)

	q, err := updateLikesQuery(Dislike, qAndAUuid)
	if err != nil {
		return fmt.Errorf("failed to like the q&a %s\n", err)
	}

	/*
	 * Resposne is deliberatly ignored because as per Cassandra Standard, inserted values do not get returned as part of response just like SQL
	 */
	_, err = cassandraClient.ExecuteQuery(q)
	if err != nil {
		return fmt.Errorf("failed to like the q&a %s\n", err)
	}
	return nil
}

func (c *CassandraLikesRepository) DeleteQAndA(ctx context.Context, qAndAUuid uuid.UUID) error {
	cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
	defer cassandraConnectionClient.Put(cassandraClient)
	cassandraClient.ExecuteBatch(&proto.Batch{
		Queries: []*proto.BatchQuery{},
	})

	cassandraCompliantQAndAUuid, err := googleUuidToCassandraUuid(qAndAUuid)
	if err != nil {
		return err
	}

	deleteQAndAQuery := `DELETE FROM main.q_and_a_likes WHERE question_id = ?`
	_, err = cassandraClient.ExecuteQuery(
		&proto.Query{
			Cql: deleteQAndAQuery,
			Values: &proto.Values{
				Values: []*proto.Value{
					&proto.Value{Inner: &proto.Value_Uuid{cassandraCompliantQAndAUuid}},
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to delete the q&a from the likes table %s", err)
	}

	return nil
}

func (c *CassandraQuestionsRepository) PostAnswerToFollowersHomefeed(context context.Context, qAndA models.QAndA, users []models.User) error {
	cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
	defer cassandraConnectionClient.Put(cassandraClient)


	postToTimeLineBatchQuery := []*proto.BatchQuery{}

	for _, u := range users {
		cassandraCompliantUuid, err := googleUuidToCassandraUuid(qAndA.QuestionId)
		if err != nil {
			return err 
		}
		postToTimeLineBatchQuery = append(postToTimeLineBatchQuery, &proto.BatchQuery{
			Cql: ` INSERT INTO main.q_and_a_followers 
			       (follower , question_id , answer , asked, asker , is_anon , question) 
				   VALUES 
				   (? , ?, ?, ?, ? , ?, ?);`,
			Values: &proto.Values{
				Values: []*proto.Value{
					&proto.Value{Inner : &proto.Value_String_{u.Username}},
					&proto.Value{Inner : &proto.Value_Uuid{cassandraCompliantUuid}},
					&proto.Value{Inner : &proto.Value_String_{qAndA.Answer}},
					&proto.Value{Inner : &proto.Value_String_{qAndA.Asked}},
					&proto.Value{Inner : &proto.Value_String_{qAndA.Asker}},
					&proto.Value{Inner : &proto.Value_Boolean{qAndA.IsAnon}},
					&proto.Value{Inner : &proto.Value_String_{qAndA.Question}},
				},
			},
		})	
	}

	if len(postToTimeLineBatchQuery) == 0 {
		return nil
	}

	_, err := cassandraClient.ExecuteBatch(&proto.Batch{Type: proto.Batch_LOGGED, Queries: postToTimeLineBatchQuery})
	if err != nil {
		return fmt.Errorf("failed to post to usertime lines %s", err)
	}

	return nil
}

func updateLikesQuery(action LikeAction, uuid uuid.UUID) (*proto.Query, error) {
	var q string
	cassandraCompliantUuid, err := googleUuidToCassandraUuid(uuid)
	if err != nil {
		return nil, err
	}

	switch action {
	case Like:
		q = `UPDATE main.q_and_a_likes SET likes = likes + 1 WHERE question_id  = ?;`
	case Dislike:
		q = `UPDATE main.q_and_a_likes SET likes = likes - 1 WHERE question_id  = ?;`
	}

	return &proto.Query{
		Cql: q,
		Values: &proto.Values{
			Values: []*proto.Value{
				&proto.Value{Inner: &proto.Value_Uuid{cassandraCompliantUuid}},
			},
		},
	}, nil
}

func cassandraUuidToGoogleUuid(v *proto.Value) (uuid.UUID, error) {
	id := v.GetUuid().Value
	parsedQuestionUuid, err := uuid.FromBytes(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to parse the uuid of one question %s\n ", err)
	}
	return parsedQuestionUuid, nil
}

func googleUuidToCassandraUuid(id uuid.UUID) (*proto.Uuid, error) {
	generatedQuestionIdInBytes, err := id.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to convert the google id to cassandra id %s", err)
	}
	cassandraCompliantQuestionUuid := &proto.Uuid{Value: generatedQuestionIdInBytes}
	return cassandraCompliantQuestionUuid, nil
}

func NewCassandraUsersRepository() UsersRepository {
	return &CassandraUsersRepository{}
}

type CassandraUsersRepository struct {
}

func (c *CassandraUsersRepository) DoesUserExist(context context.Context, u models.User) (bool, error) {
	panic("not implemented") // TODO: implement
}

func (c *CassandraUsersRepository) Register(context context.Context, u models.User) (models.User, error) {

	err := utils.ValidateRegistration(u)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to register user %s", err)
	}

	userCreationTimeUuid, err := uuid.NewUUID() 
	if err != nil {
		return models.User{}, err
	}
	cassandraCompliantUserCreationTimeUuid, err := googleUuidToCassandraUuid(userCreationTimeUuid)
	if err != nil {
		return models.User{}, err
	}
	// TODO: Password hashing?
	// hashedPassword := bcrypt.GenerateFromPassword()

	cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
	defer cassandraConnectionClient.Put(cassandraClient)

	registerUserQuery := &proto.Query{
		Cql: `INSERT INTO main.users 
				(username , created_on , email , first_name , last_name , password ) 
				VALUES (? , ? , ?, ?, ?, ?);`,
		Values: &proto.Values{
			Values: []*proto.Value{
				&proto.Value{Inner: &proto.Value_String_{u.Username}},
				&proto.Value{Inner: &proto.Value_Uuid{cassandraCompliantUserCreationTimeUuid}},
				&proto.Value{Inner: &proto.Value_String_{u.Email}},
				&proto.Value{Inner: &proto.Value_String_{u.FirstName}},
				&proto.Value{Inner: &proto.Value_String_{u.LastName}},
				&proto.Value{Inner: &proto.Value_String_{u.Password}},
			},
		},
	}
	setFollowersToZeroQuery := &proto.BatchQuery{
		Cql: `UPDATE main.followers_of_user_counter SET followers = followers + 0 WHERE username = ?;`,
		Values: &proto.Values{
			Values: []*proto.Value{
				&proto.Value{Inner: &proto.Value_String_{u.Username}},
			},
		},
	}
	setFollowingToZeroQuery := &proto.BatchQuery{
		Cql: `UPDATE main.followed_by_user_counter SET followering = followering + 0 WHERE username = ?;`,
		Values: &proto.Values{
			Values: []*proto.Value{
				&proto.Value{Inner: &proto.Value_String_{u.Username}},
			},
		},
	}

	//TODO: Use cancel here with context if any of them fail
	go func() {
		_, err := cassandraClient.ExecuteQuery(registerUserQuery)	
		if err != nil {
			panic(err)
		}
	}()

	//TODO: Use cancel here with context if any of them fail
	go func() {
		_, err := cassandraClient.ExecuteBatch(&proto.Batch{
			Type: proto.Batch_COUNTER,
			Queries:  []*proto.BatchQuery{
				setFollowersToZeroQuery,
				setFollowingToZeroQuery,
			},
		})	
		if err != nil {
			panic(err)
		}
	}()
	if err != nil {
		return models.User{}, fmt.Errorf("failed to save user in database %s", err)
	}
	return u, nil
}

func (c *CassandraUsersRepository) Login(_ context.Context, _ models.User) (models.User, error) {
	panic("not implemented") // TODO: Implement
}

func (c *CassandraUsersRepository) Delete(_ context.Context, _ models.User) error {
	panic("not implemented") // TODO: Implement
}

func (c *CassandraUsersRepository) UpdateLoginDetails(_ context.Context, _ models.User) (models.User, error) {
	panic("not implemented") // TODO: Implement
}

func (c *CassandraUsersRepository) Follow(context context.Context, follower models.User, followed models.User) error {
	panic("not implemented") // TODO: Implement
}

func (c *CassandraUsersRepository) Unfollow(context context.Context, follower models.User, followed models.User) error {
	panic("not implemented") // TODO: Implement
}

func (c *CassandraUsersRepository) SearchForUsername(context context.Context, username string) ([]models.User, error) {
	panic("not implemented") // TODO: Implement
}

func (c *CassandraUsersRepository) FindFollowersOfUser(context context.Context, username string) ([]models.User, error) {
	cassandraClient := cassandraConnectionClient.Get().(*client.StargateClient)
	defer cassandraConnectionClient.Put(cassandraClient)

	followersOfUsersQuery := `SELECT * FROM main.followers_by_user WHERE followed = ?;`

	res, err := cassandraClient.ExecuteQuery(&proto.Query{
		Cql: followersOfUsersQuery,
		Values: &proto.Values{
			Values: []*proto.Value{
				&proto.Value{Inner: &proto.Value_String_{username}},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the followers of user %s %s", username, err)
	}

	followers := []models.User{}

	for _, r := range res.GetResultSet().Rows {
		followers = append(followers, models.User{
			// The followers' column is the second
			Username: r.Values[1].GetString_(),
		})	
	}

	return followers, nil
}