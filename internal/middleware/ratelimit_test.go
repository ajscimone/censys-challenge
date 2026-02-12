package middleware

import (
	"context"
	"sync"
	"testing"
	"time"

	censysv1 "github.com/ajscimone/censys-challenge/gen/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSlidingWindowRateLimiter_AllowsUpToLimit(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(5, 1*time.Minute)

	for i := 0; i < 5; i++ {
		if !limiter.Allow("token-a") {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}

	if limiter.Allow("token-a") {
		t.Fatal("request 6 should have been denied")
	}
}

func TestSlidingWindowRateLimiter_DifferentKeysAreIndependent(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(2, 1*time.Minute)

	limiter.Allow("token-a")
	limiter.Allow("token-a")

	if limiter.Allow("token-a") {
		t.Fatal("token-a should be rate limited")
	}

	if !limiter.Allow("token-b") {
		t.Fatal("token-b should not be rate limited")
	}
}

func TestSlidingWindowRateLimiter_WindowExpiry(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(2, 50*time.Millisecond)

	limiter.Allow("token-a")
	limiter.Allow("token-a")

	if limiter.Allow("token-a") {
		t.Fatal("should be rate limited before window expires")
	}

	time.Sleep(60 * time.Millisecond)

	if !limiter.Allow("token-a") {
		t.Fatal("should be allowed after window expires")
	}
}

func TestSlidingWindowRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(100, 1*time.Minute)

	var wg sync.WaitGroup
	allowed := make(chan bool, 200)

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- limiter.Allow("token-a")
		}()
	}

	wg.Wait()
	close(allowed)

	allowedCount := 0
	for a := range allowed {
		if a {
			allowedCount++
		}
	}

	if allowedCount != 100 {
		t.Fatalf("expected exactly 100 allowed, got %d", allowedCount)
	}
}

func TestRateLimitInterceptor_BlocksSharedCollectionWhenLimited(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(1, 1*time.Minute)

	interceptor := RateLimitInterceptor(limiter)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return &censysv1.Collection{}, nil
	}

	req := &censysv1.GetSharedCollectionRequest{Token: "abc123"}

	resp, err := interceptor(context.Background(), req, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("first request should succeed: %v", err)
	}
	if resp == nil {
		t.Fatal("first request should return a response")
	}

	_, err = interceptor(context.Background(), req, &grpc.UnaryServerInfo{}, handler)
	if err == nil {
		t.Fatal("second request should be rate limited")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("error should be a gRPC status")
	}
	if st.Code() != codes.ResourceExhausted {
		t.Fatalf("expected ResourceExhausted, got %v", st.Code())
	}
}

func TestRateLimitInterceptor_PassesThroughNonSharedRequests(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(1, 1*time.Minute)

	interceptor := RateLimitInterceptor(limiter)
	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return &censysv1.Collection{}, nil
	}

	req := &censysv1.CreateCollectionRequest{Name: "test"}

	_, err := interceptor(context.Background(), req, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("non-shared request should pass through: %v", err)
	}
	if !called {
		t.Fatal("handler should have been called")
	}
}
