// Package queue is a thin Redis-backed job queue for campaign sending.
package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const launchList = "phishforge:launch"

// LaunchJob asks a worker to send a campaign.
type LaunchJob struct {
	CampaignID uuid.UUID `json:"campaign_id"`
	OrgID      uuid.UUID `json:"org_id"`
}

type Queue struct {
	rdb *redis.Client
}

func New(redisURL string) (*Queue, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &Queue{rdb: redis.NewClient(opt)}, nil
}

func (q *Queue) Ping(ctx context.Context) error { return q.rdb.Ping(ctx).Err() }
func (q *Queue) Close() error                    { return q.rdb.Close() }

func (q *Queue) EnqueueLaunch(ctx context.Context, job LaunchJob) error {
	b, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.rdb.LPush(ctx, launchList, b).Err()
}

// DequeueLaunch blocks up to timeout for the next job. Returns (nil,nil) on timeout.
func (q *Queue) DequeueLaunch(ctx context.Context, timeout time.Duration) (*LaunchJob, error) {
	res, err := q.rdb.BRPop(ctx, timeout, launchList).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(res) < 2 {
		return nil, nil
	}
	var job LaunchJob
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return nil, err
	}
	return &job, nil
}
