-- name: Create :one
INSERT INTO users (email)
VALUES ($1)
    RETURNING *;

-- name: Exists :one
SELECT id FROM users WHERE id = $1;

-- name: ExistsUser :one
SELECT EXISTS(
    select 1
    FROM users
    WHERE email = $1
);

-- name: GetByEmail :one
SELECT * FROM users WHERE email = $1;