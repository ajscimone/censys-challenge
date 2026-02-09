-- name: CreateCollection :one
INSERT INTO collections (name, data, access_level, owner_id, organization_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, uid, name, data, access_level, owner_id, organization_id, created_at, updated_at;

-- name: GetCollectionByUID :one
SELECT id, uid, name, data, access_level, owner_id, organization_id, created_at, updated_at
FROM collections
WHERE uid = $1;

-- name: GetCollectionByID :one
SELECT id, uid, name, data, access_level, owner_id, organization_id, created_at, updated_at
FROM collections
WHERE id = $1;

-- name: UpdateCollection :one
UPDATE collections
SET name = $2, data = $3, access_level = $4, organization_id = $5, updated_at = now()
WHERE id = $1
RETURNING id, uid, name, data, access_level, owner_id, organization_id, created_at, updated_at;

-- name: DeleteCollection :exec
DELETE FROM collections
WHERE id = $1;

-- name: CheckUserOwnsCollection :one
SELECT id FROM collections
WHERE id = $1 AND owner_id = $2;

-- name: CheckOrgOwnsCollection :one
SELECT c.id FROM collections c
JOIN organization_members om ON om.organization_id = c.organization_id
WHERE c.id = $1 AND om.user_id = $2 AND c.access_level = 'organization';
