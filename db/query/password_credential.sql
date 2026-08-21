-- name: CreatePasswordCredential :one
INSERT INTO password_credentials (user_id, password_hash, password_algo)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPasswordCredentialByUserID :one
SELECT * FROM password_credentials WHERE user_id = $1 LIMIT 1;

-- name: UpdatePasswordCredential :one
UPDATE password_credentials
SET password_hash = $2, password_algo = $3, password_updated_at = now(), updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeletePasswordCredential :one
DELETE FROM password_credentials WHERE id = $1 RETURNING *;

-- name: GetPasswordCredentialByEmail :one
SELECT * FROM password_credentials WHERE email = $1 LIMIT 1;