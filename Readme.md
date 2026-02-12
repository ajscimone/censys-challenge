# Censys Challenge

## Entity Relationship Diagram

```mermaid
erDiagram
    users {
        SERIAL id PK
        UUID uid UK
        TEXT email UK
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    organizations {
        SERIAL id PK
        UUID uid UK
        TEXT name
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    organization_members {
        SERIAL id PK
        INTEGER user_id FK
        INTEGER organization_id FK
        TIMESTAMPTZ created_at
    }

    collections {
        SERIAL id PK
        UUID uid UK
        TEXT name
        JSONB data
        access_level access_level
        INTEGER owner_id FK
        INTEGER organization_id FK
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    share_links {
        SERIAL id PK
        TEXT token UK
        INTEGER collection_id FK
        INTEGER access_count
        INTEGER created_by FK
        TIMESTAMPTZ created_at
    }

    organizations ||--o{ organization_members : "has"
    organization_members }o--|| users : "belongs to"
    users ||--o{ collections : "owns"
    organizations ||--o{ collections : "owns"
    collections ||--o{ share_links : "has"
    users ||--o{ share_links : "creates"
```

## Assumptions and Tradeoffs

Assumptions:
- I chose a rate limiter which limits requests to specific share links to 1000 times per 5 minutes. This is not limiting the number of requests from API users. I am making the assumption that share links are my bottle neck.
- Share links do work without authentication
- We track who accesses each collection via normal auth, but anyone with a share link token can gain access to them so no way to trace those accesses for security. There are other ways we could track this with something like an audit logger or a prometheus stream.
- updates to share links are done at the database level which is very near real time but not as close to real time as something like an in memory cache might be.

Tradeoffs: 
- using a rate limiter over other options (caching, request coalescing, cdn)
- lots of implmentations are using direct database calls and concrete types done for the sake of time. This will make the implementation difficult to test and difficult to trade out packages and function calls for alternative implementations

Short cuts:
- organization names not unique
- Login does no real authentication for the purpose of simplifying the challenge
- Revocation happens at the database layer as opposed to something higher up the stack
- Rate limiter is limiting on calls to individual share tokens per share token as opposed to total requests or ip addresses
- JWT auth does not currently expire tokens for the sake of simplicity. We only check that we signed it
- I chose to put the Login method inside the Collections service since we have simplified auth for this challenge and dont have an auth service


## Setup

### Docker

To run everything you can run:
`docker compose up -d`

To kill all the containers and delete you can run
`docker compose down -v`

A quick workflow is 
`docker compose down -v && docker-compose build --no-cache app && docker-compose up -d`

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