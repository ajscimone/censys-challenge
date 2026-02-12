package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajscimone/censys-challenge/internal/authentication"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const claimsKey contextKey = "claims"

func AuthInterceptor(auth *authentication.Authenticator) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		skipMethods := map[string]bool{
			"/censys.v1.CollectionService/Login":               true,
			"/censys.v1.CollectionService/GetSharedCollection": true,
			"/censys.v1.AdminService/CreateUser":               true,
			"/censys.v1.AdminService/CreateOrganization":       true,
			"/censys.v1.AdminService/AddOrganizationMember":    true,
		}

		if skipMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		token := authHeader[0]
		if !strings.HasPrefix(token, "Bearer ") {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
		}
		token = strings.TrimPrefix(token, "Bearer ")

		claims, err := auth.ValidateToken(token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		ctx = context.WithValue(ctx, claimsKey, claims)

		return handler(ctx, req)
	}
}

func UserIDFromContext(ctx context.Context) (int32, error) {
	claims, ok := ctx.Value(claimsKey).(*authentication.Claims)
	if !ok {
		return 0, fmt.Errorf("no authentication claims in context")
	}
	return claims.UserID, nil
}
