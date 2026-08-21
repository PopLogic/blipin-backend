-- name: CreateSession :one
INSERT INTO sessions (
    id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetSessionByID :one
SELECT * FROM sessions WHERE id = $1 LIMIT 1;

-- name: UpdateSessionStatus :one
UPDATE sessions
SET is_blocked = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateSessionRefreshTokenAndExpiredAt :one
UPDATE sessions
SET refresh_token = $2, expires_at = $3, updated_at = now()
WHERE id = $1
RETURNING *;