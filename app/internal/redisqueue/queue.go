package redisqueue

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const DiscoveredJobsKey = "jobs:discovered"

type Queue struct {
	client *redis.Client
}

func New(redisURL string) (*Queue, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &Queue{client: redis.NewClient(opts)}, nil
}

func (q *Queue) Ping(ctx context.Context) error {
	return q.client.Ping(ctx).Err()
}

func (q *Queue) Push(ctx context.Context, key, value string) error {
	return q.client.LPush(ctx, key, value).Err()
}

// Pop blocks until an item is available or the context is cancelled.
func (q *Queue) Pop(ctx context.Context, key string) (string, error) {
	res, err := q.client.BRPop(ctx, 0, key).Result()
	if err != nil {
		return "", err
	}
	// BRPop returns [key, value]
	return res[1], nil
}

func (q *Queue) Close() error {
	return q.client.Close()
}
