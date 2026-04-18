package contextw

import (
	"context"
)

type ContextValue[V any] string

func (cv ContextValue[V]) Set(ctx context.Context, value V) context.Context {
	return context.WithValue(ctx, cv, value)
}
func (cv ContextValue[V]) Get(ctx context.Context) (V, bool) {
	var emptyValue V
	value, ok := ctx.Value(cv).(V)
	if !ok {
		return emptyValue, false
	}
	return value, true
}
