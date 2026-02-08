-- name: GetAllUsers :many
SELECT id, uid, email, name, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- name: GetUserByID :one
SELECT id, uid, email, name, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, uid, email, name, created_at, updated_at
FROM users
WHERE email = $1;
