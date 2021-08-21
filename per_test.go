package per

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRequireRecompute(t *testing.T) {
	const key, value, ttl = "key", "value", time.Second
	ctx := context.Background()
	compute := func(ctx context.Context) (interface{}, error) {
		return value, nil
	}
	computeErr := func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("error")
	}

	c := New()
	ret, err := c.Get(ctx, key, ttl, compute)
	require.NoError(t, err)
	require.Equal(t, value, ret)

	ret, err = c.Get(ctx, "error", ttl, computeErr)
	require.Error(t, err)

	items := c.BatchGetItems(ctx, []string{key})
	require.Equal(t, 1, len(items))
	c.BatchSetItems(ctx, items, 0)
}

func TestItems(t *testing.T) {
	items := Items{{Value: "foo"}}
	require.Equal(t, []interface{}{items[0]}, items.Interfaces())
	require.Equal(t, []interface{}{"foo"}, items.Values())
}
