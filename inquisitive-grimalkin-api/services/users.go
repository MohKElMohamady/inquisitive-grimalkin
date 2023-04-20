package services

import (
	"context"
	"inquisitive-grimalkin/data"
)

type UsersService struct {
	userRepostory data.CassandraUsersRepository
}


func (s *UsersService) Follow(context context.Context, follower string, following string) error {
	s.userRepostory.Follow(context, follower, following)
	return nil
}

func (s *UsersService) Unfollow(context context.Context, follower string, following string) error {
	s.userRepostory.Follow(context, follower, following)
	return nil
}