package utils

import "context"

type ctxKey int


const (
	userCtxKey ctxKey = iota
	
)

func ContextWithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, userCtxKey, username)
}

func UserFromContext(context context.Context) (string, bool) {
	user, ok := context.Value(userCtxKey).(string)
	return user, ok	
}