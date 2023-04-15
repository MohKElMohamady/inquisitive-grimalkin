package data

import (
	"context"
	"crypto/tls"
	"inquisitive-grimalkin/models"
	"log"
	"os"
	"sync"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stargate/stargate-grpc-go-client/stargate/pkg/auth"
	"github.com/stargate/stargate-grpc-go-client/stargate/pkg/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var cassandraRemoteUri = ""
var cassandraClientId = ""
var cassandraClientSecret = ""
var cassandraBearerToken = ""

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
		log.Fatal("something went wrong in loading the variables")
	}
	cassandraRemoteUri = os.Getenv("CASSANDRA_REMOTE_URI") 
	cassandraClientId = os.Getenv("CASSANDRA_CLIENT_ID") 
	cassandraClientSecret = os.Getenv("CASSANDRA_CLIENT_SECRET") 
	cassandraBearerToken = os.Getenv("CASSANDRA_BEARER_TOKEN") 
}
