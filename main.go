package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

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

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
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
