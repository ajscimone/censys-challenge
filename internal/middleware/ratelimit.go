package middleware

import (
	"context"
	"sync"
	"time"

	censysv1 "github.com/ajscimone/censys-challenge/gen/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RateLimiter interface {
	Allow(key string) bool
}

type entry struct {
	timestamps []time.Time
}

type SlidingWindowRateLimiter struct {
	mu     sync.Mutex
	keys   map[string]*entry
	limit  int
	window time.Duration
}

func NewSlidingWindowRateLimiter(limit int, window time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		keys:   make(map[string]*entry),
		limit:  limit,
		window: window,
	}
}

func (s *SlidingWindowRateLimiter) Allow(key string) bool {
	// this would be better as a read lock
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-s.window)

	e, exists := s.keys[key]
	if !exists {
		s.keys[key] = &entry{timestamps: []time.Time{now}}
		return true
	}

	// this is naive and could become a bottle neck. It would be better to have a goroutine that is constantly pruning
	pruned := make([]time.Time, 0, len(e.timestamps))
	for _, ts := range e.timestamps {
		if ts.After(cutoff) {
			pruned = append(pruned, ts)
		}
	}

	if len(pruned) >= s.limit {
		e.timestamps = pruned
		return false
	}

	e.timestamps = append(pruned, now)
	return true
}

func RateLimitInterceptor(limiter RateLimiter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		sharedReq, ok := req.(*censysv1.GetSharedCollectionRequest)
		if !ok {
			return handler(ctx, req)
		}

		if !limiter.Allow(sharedReq.Token) {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}
