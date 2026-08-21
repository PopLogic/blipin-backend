-- name: CreateUser :one
INSERT INTO users (display_name, avatar_url, status)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteUser :one
UPDATE users
SET deleted_at = now(), updated_at = now(), status = 'deleted'
WHERE id = $1
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
UPDATE users
SET display_name = $2, birthdate = $3, gender = $4, updated_at = now()
WHERE id = $1
RETURNING *;