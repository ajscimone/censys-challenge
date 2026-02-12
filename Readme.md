# Censys Challenge

## Setup

### Docker

To run everything you can run:
`docker compose up -d`

To kill all the containers and delete you can run
`docker-compose down -v`

A quick workflow is 
`docker-compose down -v && docker-compose build --no-cache app && docker-compose up -d`

### Migrations

To do things more manually, running migrations:
`migrate -path db/migrations -database "postgres://admin:password1@localhost:5432/censys-challenge?sslmode=disable" up`

### Generate sqlc

`sqlc generate`

### Generate Proto

`buf generate`

### Running

Start the service without docker: `go run main.go`
I am using go 1.25 but it likely works with lower versions.

## Manual Testing with grpcurl

### 1. Setup Test Data

Create users:
```bash
grpcurl -plaintext -d '{"email":"tony@example.com"}' localhost:50051 censys.v1.AdminService/CreateUser
```

Create organization:
```bash
grpcurl -plaintext -d '{"name":"Example"}' localhost:50051 censys.v1.AdminService/CreateOrganization
```

Add user to organization (use the UIDs from responses above):
```bash
grpcurl -plaintext -d '{"user_uid":"<user_uid>","organization_uid":"<org_uid>"}' localhost:50051 censys.v1.AdminService/AddOrganizationMember
```

### 2. Authentication

Login as user:
```bash
grpcurl -plaintext -d '{"email":"tony@example.com"}' localhost:50051 censys.v1.CollectionService/Login
```

### 3. Create Collections

Create a private collection:
```bash
grpcurl -plaintext -H "authorization: Bearer $TOKEN1" -d '{"name":"My Private Collection","access_level":"ACCESS_LEVEL_PRIVATE","data":{"type":"saved_search","query":"test"}}' localhost:50051 censys.v1.CollectionService/CreateCollection
```

Create an organization collection:
```bash
grpcurl -plaintext -H "authorization: Bearer $TOKEN1" -d '{"name":"Org Shared Collection","access_level":"ACCESS_LEVEL_ORGANIZATION","organization_uid":"<org_uid>","data":{"type":"dataset"}}' localhost:50051 censys.v1.CollectionService/CreateCollection
```

### 4. Get Collections

Get a collection :
```bash
grpcurl -plaintext -H "authorization: Bearer $TOKEN1" -d '{"uid":"<private_collection_uid>"}' localhost:50051 censys.v1.CollectionService/GetCollection
```

### 5. Share Tokens

Create share token for private collection:
```bash
grpcurl -plaintext -H "authorization: Bearer $TOKEN1" -d '{"collection_uid":"<private_collection_uid>"}' localhost:50051 censys.v1.CollectionService/CreateShareToken
```

Access collection via share token:
```bash
grpcurl -plaintext -d '{"token":"<share_token>"}' localhost:50051 censys.v1.CollectionService/GetSharedCollection
```

### 6. Update Collection

Update collection name:
```bash
grpcurl -plaintext -H "authorization: Bearer $TOKEN1" -d '{"uid":"<private_collection_uid>","name":"Updated Private Collection"}' localhost:50051 censys.v1.CollectionService/UpdateCollection
```

### 7. Revoke Share Token

Revoke the share token:
```bash
grpcurl -plaintext -H "authorization: Bearer $TOKEN1" -d '{"token":"<share_token>"}' localhost:50051 censys.v1.CollectionService/RevokeShareToken
```

### 8. Delete Collection

Delete collection:
```bash
grpcurl -plaintext -H "authorization: Bearer $TOKEN1" -d '{"uid":"<private_collection_uid>"}' localhost:50051 censys.v1.CollectionService/DeleteCollection
```

### 9. List Available Services

```bash
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50051 describe censys.v1.CollectionService
```