package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ajscimone/censys-challenge/gen/proto"
	"github.com/ajscimone/censys-challenge/internal/authentication"
	"github.com/ajscimone/censys-challenge/internal/db"
	"github.com/ajscimone/censys-challenge/internal/server"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setUpOrg(queries *db.Queries, ctx context.Context) {
	adminServer := server.NewAdminServer(queries)

	user, err := adminServer.CreateUser(ctx, &censysv1.CreateUserRequest{
		Email: "tony@example.com",
	})
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	fmt.Printf("Created user: UID=%s, Email=%s\n", user.Uid, user.Email)

	org, err := adminServer.CreateOrganization(ctx, &censysv1.CreateOrganizationRequest{
		Name: "Test Org",
	})
	if err != nil {
		log.Fatalf("Failed to create organization: %v", err)
	}
	fmt.Printf("Created org: UID=%s, Name=%s\n", org.Uid, org.Name)

	err = adminServer.AddOrganizationMember(ctx, &censysv1.AddOrganizationMemberRequest{
		UserUid:         user.Uid,
		OrganizationUid: org.Uid,
	})
	if err != nil {
		log.Fatalf("Failed to add member: %v", err)
	}
}

func main() {
	ctx := context.Background()

	dbURL := "postgres://admin:password1@localhost:5432/censys-challenge?sslmode=disable"
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	auth := authentication.NewAuthenticator(queries, "salami-is-bomb")

	token, err := auth.Login(ctx, "tony@example.com")
	if err != nil {
		log.Fatalf("Failed to login: %v", err)
	}
	fmt.Printf("Generated JWT: %s\n", token)

	_, err = auth.ValidateToken(token)
	if err != nil {
		log.Fatalf("Failed to validate token: %v", err)
	}

	_, err = auth.ValidateToken("invalid-token-here")
	if err != nil {
		fmt.Printf("Mission failed successfully: %v\n", err)
	}
}