package middleware_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/silviolleite/loafer-natsx/middleware"
)

func record(order *[]string, name string) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, data []byte) (any, error) {
			*order = append(*order, name+":in")
			res, err := next(ctx, data)
			*order = append(*order, name+":out")
			return res, err
		}
	}
}

func TestChain_Order(t *testing.T) {
	var order []string

	base := func(ctx context.Context, data []byte) (any, error) {
		order = append(order, "handler")
		return "ok", nil
	}

	chained := middleware.Chain(
		record(&order, "A"),
		record(&order, "B"),
		record(&order, "C"),
	)(base)

	res, err := chained(context.Background(), nil)

	assert.NoError(t, err)
	assert.Equal(t, "ok", res)
	assert.Equal(t, []string{
		"A:in", "B:in", "C:in",
		"handler",
		"C:out", "B:out", "A:out",
	}, order)
}

func TestChain_SkipsNil(t *testing.T) {
	var order []string

	base := func(ctx context.Context, data []byte) (any, error) {
		order = append(order, "handler")
		return nil, nil
	}

	chained := middleware.Chain(nil, record(&order, "A"), nil)(base)

	_, err := chained(context.Background(), nil)

	assert.NoError(t, err)
	assert.Equal(t, []string{"A:in", "handler", "A:out"}, order)
}

func TestChain_Empty(t *testing.T) {
	called := false

	base := func(ctx context.Context, data []byte) (any, error) {
		called = true
		return nil, nil
	}

	chained := middleware.Chain()(base)

	_, err := chained(context.Background(), nil)

	assert.NoError(t, err)
	assert.True(t, called)
}
