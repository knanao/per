package per

import (
	"context"
	"math"
	"math/rand"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

type ComputeFunc func(ctx context.Context) (interface{}, error)

type Item struct {
	Key    string
	Value  interface{}
	Delta  time.Duration
	Expiry time.Time
}

type Items []*Item

func (i *Item) Recompute(now time.Time) bool {
	const beta float64 = 1
	if i == nil {
		return true
	}
	v := i.deltaNs() * beta * math.Log(rand.Float64()) // nolint:gosec
	return now.Add(time.Duration(-v)).Sub(i.Expiry) >= 0
}

func (i *Item) deltaNs() float64 {
	return float64(i.Delta.Nanoseconds())
}

func (i Items) Interfaces() []interface{} {
	results := make([]interface{}, len(i))
	for j, item := range i {
		results[j] = item
	}
	return results
}

func (i Items) Values() []interface{} {
	results := make([]interface{}, len(i))
	for j, item := range i {
		results[j] = item.Value
	}
	return results
}

type Cache interface {
	Get(ctx context.Context, key string, ttl time.Duration, f ComputeFunc) (interface{}, error)
	SetItem(ctx context.Context, item Item, ttl time.Duration)
	BatchGetItems(ctx context.Context, keys []string) Items
	BatchSetItems(ctx context.Context, items Items, ttl time.Duration)
}

type cache struct {
	c   *gocache.Cache
	now func() time.Time
}

func New() Cache {
	return &cache{
		c:   gocache.New(time.Minute, time.Minute),
		now: time.Now,
	}
}

func (c *cache) BatchSetItems(_ context.Context, items Items, ttl time.Duration) {
	for i := range items {
		c.c.Set(items[i].Key, items[i], ttl)
	}
}

func (c *cache) BatchGetItems(ctx context.Context, keys []string) Items {
	values := make(Items, 0, len(keys))
	for i := range keys {
		v, ok := c.get(keys[i])
		if ok {
			values = append(values, v)
		}
	}
	return values
}

func (c *cache) SetItem(_ context.Context, item Item, ttl time.Duration) {
	c.c.Set(item.Key, item, ttl)
}

func (c *cache) Get(ctx context.Context, key string, ttl time.Duration, f ComputeFunc) (interface{}, error) {
	start := c.now()
	item, ok := c.get(key)
	if !ok {
		v, err := f(ctx)
		latency := time.Since(start)
		if err != nil {
			return nil, err
		}
		item := &Item{
			Key:    key,
			Value:  v,
			Delta:  latency,
			Expiry: start.Add(ttl),
		}
		c.c.Set(key, item, ttl)
		return v, nil
	}
	isRecompute := item.Recompute(start)
	if isRecompute {
		go func() {
			v, err := f(ctx)
			latency := time.Since(start)
			if err != nil {
			}
			item := &Item{
				Key:    key,
				Value:  v,
				Delta:  latency,
				Expiry: start.Add(ttl),
			}
			c.c.Set(key, item, ttl)
		}()
	}
	return item.Value, nil
}

func (c *cache) get(key string) (*Item, bool) {
	v, ok := c.c.Get(key)
	if !ok {
		return nil, false
	}
	return v.(*Item), true
}
