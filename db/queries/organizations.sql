-- name: IsUserInOrganization :one
SELECT id FROM organization_members
WHERE user_id = $1 AND organization_id = $2;

-- name: GetOrganizationByUID :one
SELECT id, uid, name, created_at, updated_at
FROM organizations
WHERE uid = $1;

-- name: CreateOrganization :one
INSERT INTO organizations (name)
VALUES ($1)
RETURNING id, uid, name, created_at, updated_at;

-- name: AddOrganizationMember :exec
INSERT INTO organization_members (user_id, organization_id)
VALUES ($1, $2);
