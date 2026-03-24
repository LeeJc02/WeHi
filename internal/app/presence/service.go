package presence

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// onlineSetKey stores the coarse-grained online presence view used by delivery
// decisions and admin diagnostics; connection fan-out still lives in realtime.Hub.
const onlineSetKey = "presence:online"

type Service struct {
	redis *redis.Client
}

func NewService(redisClient *redis.Client) *Service {
	return &Service{redis: redisClient}
}

// MarkOnline and MarkOffline intentionally keep presence state small and
// writable at high frequency instead of turning MySQL into a heartbeat sink.
func (s *Service) MarkOnline(ctx context.Context, userID uint64) error {
	return s.redis.SAdd(ctx, onlineSetKey, userID).Err()
}

func (s *Service) MarkOffline(ctx context.Context, userID uint64) error {
	return s.redis.SRem(ctx, onlineSetKey, userID).Err()
}

// OnlineMap batches membership checks in one pipeline so message delivery logic
// can classify recipients without issuing one round-trip per user.
func (s *Service) OnlineMap(ctx context.Context, userIDs []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	pipe := s.redis.Pipeline()
	checks := make([]*redis.BoolCmd, 0, len(userIDs))
	for _, userID := range userIDs {
		checks = append(checks, pipe.SIsMember(ctx, onlineSetKey, userID))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	for idx, userID := range userIDs {
		value, err := checks[idx].Result()
		if err != nil {
			return nil, err
		}
		result[userID] = value
	}
	return result, nil
}

func (s *Service) OnlineUsers(ctx context.Context) ([]uint64, error) {
	members, err := s.redis.SMembers(ctx, onlineSetKey).Result()
	if err != nil {
		return nil, err
	}
	result := make([]uint64, 0, len(members))
	for _, member := range members {
		value, err := strconv.ParseUint(member, 10, 64)
		if err != nil {
			continue
		}
		result = append(result, value)
	}
	return result, nil
}
