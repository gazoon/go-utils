package request

import (
	"context"
	"github.com/satori/go.uuid"
)

type ContextKey int

var (
	requestIdCtxKey = ContextKey(1)
)

func NewRequestId() string {
	return uuid.NewV4().String()
}

func NewContext(ctx context.Context, requestId string) context.Context {
	return context.WithValue(ctx, requestIdCtxKey, requestId)
}

func NewContextBackground(requestId string) context.Context {
	return NewContext(context.Background(), requestId)
}

func FromContext(ctx context.Context) string {
	requestId, _ := ctx.Value(requestIdCtxKey).(string)
	return requestId
}
