-- name: CreateShareLink :one
INSERT INTO share_links (token, collection_id, created_by)
VALUES ($1, $2, $3)
RETURNING id, token, collection_id, access_count, created_by, created_at;

-- name: GetShareLinkByToken :one
SELECT id, token, collection_id, access_count, created_by, created_at
FROM share_links
WHERE token = $1;

-- name: IncrementAccessCount :one
UPDATE share_links
SET access_count = access_count + 1
WHERE token = $1
RETURNING id, token, collection_id, access_count, created_by, created_at;

-- name: DeleteShareLinkByToken :exec
DELETE FROM share_links
WHERE token = $1;

-- name: GetShareLinksByCollectionID :many
SELECT id, token, collection_id, access_count, created_by, created_at
FROM share_links
WHERE collection_id = $1;
