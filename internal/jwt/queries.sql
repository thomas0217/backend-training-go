-- name: Create :one
INSERT INTO refresh_tokens (user_id, expiration_time)
VALUES ($1, $2)
    RETURNING *;

-- name: Get :one
SELECT * FROM refresh_tokens WHERE id = $1;

-- name: Use :one
UPDATE refresh_tokens
SET is_available = false
WHERE id = $1 AND is_available = true AND expiration_time > now()
    RETURNING *;
