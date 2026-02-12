package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ajscimone/censys-challenge/gen/proto"
	"github.com/ajscimone/censys-challenge/internal/authentication"
	"github.com/ajscimone/censys-challenge/internal/db"
	"github.com/ajscimone/censys-challenge/internal/middleware"
	"github.com/ajscimone/censys-challenge/internal/server"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	ctx := context.Background()

	// would be better to use a repository pattern around the database here so that database can be swapped out for things like an in memory implementation
	dbURL := getEnv("DATABASE_URL", "postgres://admin:password1@localhost:5432/censys-challenge?sslmode=disable")
	jwtSecret := getEnv("JWT_SECRET", "salami-is-bomb")
	port := getEnv("PORT", "50051")

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	auth := authentication.NewAuthenticator(queries, jwtSecret)

	rateLimiter := middleware.NewSlidingWindowRateLimiter(1000, 5*time.Minute)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RateLimitInterceptor(rateLimiter),
			middleware.AuthInterceptor(auth),
		),
	)

	censysv1.RegisterCollectionServiceServer(grpcServer, server.NewCollectionServer(queries, auth))
	censysv1.RegisterAdminServiceServer(grpcServer, server.NewAdminServer(queries))

	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("gRPC server listening on :%s", port)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	grpcServer.GracefulStop()
}
