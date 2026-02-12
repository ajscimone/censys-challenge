package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ajscimone/censys-challenge/gen/proto"
	"github.com/ajscimone/censys-challenge/internal/authentication"
	"github.com/ajscimone/censys-challenge/internal/db"
	"github.com/ajscimone/censys-challenge/internal/middleware"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CollectionServer struct {
	censysv1.UnimplementedCollectionServiceServer
	queries *db.Queries
	auth    *authentication.Authenticator
}

func NewCollectionServer(queries *db.Queries, auth *authentication.Authenticator) *CollectionServer {
	return &CollectionServer{
		queries: queries,
		auth:    auth,
	}
}

func (s *CollectionServer) CreateCollection(ctx context.Context, req *censysv1.CreateCollectionRequest) (*censysv1.Collection, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	userID, err := middleware.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	var orgID pgtype.Int4
	if req.AccessLevel == censysv1.AccessLevel_ACCESS_LEVEL_ORGANIZATION {
		if req.OrganizationUid == "" {
			return nil, status.Error(codes.InvalidArgument, "organization_uid required for organization-level access")
		}

		var orgUUID pgtype.UUID
		if err := orgUUID.Scan(req.OrganizationUid); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid organization_uid: %v", err)
		}

		org, err := s.queries.GetOrganizationByUID(ctx, orgUUID)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "organization not found: %v", err)
		}

		_, err = s.queries.IsUserInOrganization(ctx, db.IsUserInOrganizationParams{
			UserID:         userID,
			OrganizationID: org.ID,
		})
		if err != nil {
			return nil, status.Error(codes.PermissionDenied, "user not in organization")
		}

		orgID = pgtype.Int4{Int32: org.ID, Valid: true}
	}

	dataBytes, err := req.Data.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid data: %v", err)
	}

	dbCollection, err := s.queries.CreateCollection(ctx, db.CreateCollectionParams{
		Name:           req.Name,
		Data:           dataBytes,
		AccessLevel:    db.AccessLevel(req.AccessLevel.String()[len("ACCESS_LEVEL_"):]),
		OwnerID:        pgtype.Int4{Int32: userID, Valid: true},
		OrganizationID: orgID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create collection: %v", err)
	}

	return dbCollectionToProto(dbCollection)
}

func (s *CollectionServer) GetCollection(ctx context.Context, req *censysv1.GetCollectionRequest) (*censysv1.Collection, error) {
	if req.Uid == "" {
		return nil, status.Error(codes.InvalidArgument, "uid is required")
	}

	var collectionUUID pgtype.UUID
	if err := collectionUUID.Scan(req.Uid); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid uid: %v", err)
	}

	dbCollection, err := s.queries.GetCollectionByUID(ctx, collectionUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "collection not found: %v", err)
	}

	userID, err := middleware.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	if !checkAccess(ctx, s.queries, dbCollection.ID, userID) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	return dbCollectionToProto(dbCollection)
}

func (s *CollectionServer) UpdateCollection(ctx context.Context, req *censysv1.UpdateCollectionRequest) (*censysv1.Collection, error) {
	if req.Uid == "" {
		return nil, status.Error(codes.InvalidArgument, "uid is required")
	}

	var collectionUUID pgtype.UUID
	if err := collectionUUID.Scan(req.Uid); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid uid: %v", err)
	}

	dbCollection, err := s.queries.GetCollectionByUID(ctx, collectionUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "collection not found: %v", err)
	}

	userID, err := middleware.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	if !checkAccess(ctx, s.queries, dbCollection.ID, userID) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	name := dbCollection.Name
	if req.Name != "" {
		name = req.Name
	}

	dataBytes := dbCollection.Data
	if req.Data != nil && len(req.Data.Fields) > 0 {
		var err error
		dataBytes, err = req.Data.MarshalJSON()
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid data: %v", err)
		}
	}

	accessLevel := dbCollection.AccessLevel
	orgID := dbCollection.OrganizationID
	if req.AccessLevel != censysv1.AccessLevel_ACCESS_LEVEL_UNSPECIFIED {
		accessLevel = db.AccessLevel(req.AccessLevel.String()[len("ACCESS_LEVEL_"):])

		if req.AccessLevel == censysv1.AccessLevel_ACCESS_LEVEL_ORGANIZATION {
			if req.OrganizationUid == "" {
				return nil, status.Error(codes.InvalidArgument, "organization_uid required for organization-level access")
			}

			var orgUUID pgtype.UUID
			if err := orgUUID.Scan(req.OrganizationUid); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid organization_uid: %v", err)
			}

			org, err := s.queries.GetOrganizationByUID(ctx, orgUUID)
			if err != nil {
				return nil, status.Errorf(codes.NotFound, "organization not found: %v", err)
			}

			orgID = pgtype.Int4{Int32: org.ID, Valid: true}
		} else {
			orgID = pgtype.Int4{Valid: false}
		}
	}

	updated, err := s.queries.UpdateCollection(ctx, db.UpdateCollectionParams{
		ID:             dbCollection.ID,
		Name:           name,
		Data:           dataBytes,
		AccessLevel:    accessLevel,
		OrganizationID: orgID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update collection: %v", err)
	}

	return dbCollectionToProto(updated)
}

func (s *CollectionServer) DeleteCollection(ctx context.Context, req *censysv1.DeleteCollectionRequest) (*emptypb.Empty, error) {
	if req.Uid == "" {
		return nil, status.Error(codes.InvalidArgument, "uid is required")
	}

	var collectionUUID pgtype.UUID
	if err := collectionUUID.Scan(req.Uid); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid uid: %v", err)
	}

	dbCollection, err := s.queries.GetCollectionByUID(ctx, collectionUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "collection not found: %v", err)
	}

	userID, err := middleware.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	if !checkAccess(ctx, s.queries, dbCollection.ID, userID) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	if err := s.queries.DeleteCollection(ctx, dbCollection.ID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete collection: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func checkAccess(ctx context.Context, queries *db.Queries, collectionID, userID int32) bool {
	_, err := queries.CheckUserOwnsCollection(ctx, db.CheckUserOwnsCollectionParams{
		ID:      collectionID,
		OwnerID: pgtype.Int4{Int32: userID, Valid: true},
	})
	if err == nil {
		return true
	}

	_, err = queries.CheckOrgOwnsCollection(ctx, db.CheckOrgOwnsCollectionParams{
		ID:     collectionID,
		UserID: userID,
	})
	return err == nil
}

func dbCollectionToProto(c db.Collection) (*censysv1.Collection, error) {
	uidBytes, err := c.Uid.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal uid: %v", err)
	}

	var dataStruct structpb.Struct
	if err := json.Unmarshal(c.Data, &dataStruct); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal data: %v", err)
	}

	var accessLevel censysv1.AccessLevel
	switch c.AccessLevel {
	case db.AccessLevelPrivate:
		accessLevel = censysv1.AccessLevel_ACCESS_LEVEL_PRIVATE
	case db.AccessLevelOrganization:
		accessLevel = censysv1.AccessLevel_ACCESS_LEVEL_ORGANIZATION
	case db.AccessLevelShared:
		accessLevel = censysv1.AccessLevel_ACCESS_LEVEL_SHARED
	default:
		accessLevel = censysv1.AccessLevel_ACCESS_LEVEL_UNSPECIFIED
	}

	ownerID := ""
	if c.OwnerID.Valid {
		ownerID = fmt.Sprintf("%d", c.OwnerID.Int32)
	}

	orgID := ""
	if c.OrganizationID.Valid {
		orgID = fmt.Sprintf("%d", c.OrganizationID.Int32)
	}

	return &censysv1.Collection{
		Uid:            string(uidBytes[1 : len(uidBytes)-1]),
		Name:           c.Name,
		Data:           &dataStruct,
		AccessLevel:    accessLevel,
		OwnerId:        ownerID,
		OrganizationId: orgID,
		CreatedAt:      timestamppb.New(c.CreatedAt.Time),
		UpdatedAt:      timestamppb.New(c.UpdatedAt.Time),
	}, nil
}

func (s *CollectionServer) CreateShareToken(ctx context.Context, req *censysv1.CreateShareTokenRequest) (*censysv1.ShareToken, error) {
	if req.CollectionUid == "" {
		return nil, status.Error(codes.InvalidArgument, "collection_uid is required")
	}

	userID, err := middleware.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	var collectionUUID pgtype.UUID
	if err := collectionUUID.Scan(req.CollectionUid); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid collection_uid: %v", err)
	}

	dbCollection, err := s.queries.GetCollectionByUID(ctx, collectionUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "collection not found: %v", err)
	}

	if !checkAccess(ctx, s.queries, dbCollection.ID, userID) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	token, err := generateSecureToken()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate token: %v", err)
	}

	shareLink, err := s.queries.CreateShareLink(ctx, db.CreateShareLinkParams{
		Token:        token,
		CollectionID: dbCollection.ID,
		CreatedBy:    userID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create share link: %v", err)
	}

	collectionUIDBytes, err := dbCollection.Uid.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal collection uid: %v", err)
	}

	return &censysv1.ShareToken{
		Token:         shareLink.Token,
		CollectionUid: string(collectionUIDBytes[1 : len(collectionUIDBytes)-1]),
		AccessCount:   shareLink.AccessCount,
		CreatedAt:     timestamppb.New(shareLink.CreatedAt.Time),
	}, nil
}

func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *CollectionServer) GetSharedCollection(ctx context.Context, req *censysv1.GetSharedCollectionRequest) (*censysv1.SharedCollectionResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	shareLink, err := s.queries.IncrementAccessCount(ctx, req.Token)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "invalid or revoked token: %v", err)
	}

	dbCollection, err := s.queries.GetCollectionByID(ctx, shareLink.CollectionID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "collection not found: %v", err)
	}

	protoCollection, err := dbCollectionToProto(dbCollection)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert collection: %v", err)
	}

	return &censysv1.SharedCollectionResponse{
		Collection:  protoCollection,
		AccessCount: shareLink.AccessCount,
	}, nil

}

// This is purely to simplify the challenge to expose a login method through rpc.
func (s *CollectionServer) Login(ctx context.Context, req *censysv1.LoginRequest) (*censysv1.LoginResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	token, err := s.auth.Login(ctx, req.Email)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "login failed: %v", err)
	}

	return &censysv1.LoginResponse{
		Token: token,
	}, nil
}
