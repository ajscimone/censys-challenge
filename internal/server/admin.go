package server

import (
	"context"

	"github.com/ajscimone/censys-challenge/gen/proto"
	"github.com/ajscimone/censys-challenge/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AdminServer struct {
	censysv1.UnimplementedAdminServiceServer
	queries *db.Queries
}

func NewAdminServer(queries *db.Queries) *AdminServer {
	return &AdminServer{
		queries: queries,
	}
}

func (s *AdminServer) CreateUser(ctx context.Context, req *censysv1.CreateUserRequest) (*censysv1.User, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	dbUser, err := s.queries.CreateUser(ctx, req.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	uidBytes, err := dbUser.Uid.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal uid: %v", err)
	}

	return &censysv1.User{
		Uid:   string(uidBytes[1 : len(uidBytes)-1]),
		Email: dbUser.Email,
	}, nil
}

func (s *AdminServer) CreateOrganization(ctx context.Context, req *censysv1.CreateOrganizationRequest) (*censysv1.Organization, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	dbOrg, err := s.queries.CreateOrganization(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create organization: %v", err)
	}

	uidBytes, err := dbOrg.Uid.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal uid: %v", err)
	}

	return &censysv1.Organization{
		Uid:  string(uidBytes[1 : len(uidBytes)-1]),
		Name: dbOrg.Name,
	}, nil
}

func (s *AdminServer) AddOrganizationMember(ctx context.Context, req *censysv1.AddOrganizationMemberRequest) (*censysv1.OrganizationMembership, error) {
	if req.UserUid == "" {
		return nil, status.Error(codes.InvalidArgument, "user_uid is required")
	}
	if req.OrganizationUid == "" {
		return nil, status.Error(codes.InvalidArgument, "organization_uid is required")
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(req.UserUid); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_uid: %v", err)
	}

	var orgUUID pgtype.UUID
	if err := orgUUID.Scan(req.OrganizationUid); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization_uid: %v", err)
	}

	dbUser, err := s.queries.GetUserByUID(ctx, userUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	dbOrg, err := s.queries.GetOrganizationByUID(ctx, orgUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "organization not found: %v", err)
	}

	err = s.queries.AddOrganizationMember(ctx, db.AddOrganizationMemberParams{
		UserID:         dbUser.ID,
		OrganizationID: dbOrg.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add member: %v", err)
	}

	userUIDBytes, err := dbUser.Uid.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal user uid: %v", err)
	}

	orgUIDBytes, err := dbOrg.Uid.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal org uid: %v", err)
	}

	return &censysv1.OrganizationMembership{
		User: &censysv1.User{
			Uid:   string(userUIDBytes[1 : len(userUIDBytes)-1]),
			Email: dbUser.Email,
		},
		Organization: &censysv1.Organization{
			Uid:  string(orgUIDBytes[1 : len(orgUIDBytes)-1]),
			Name: dbOrg.Name,
		},
	}, nil
}
