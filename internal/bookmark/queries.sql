-- name: GetFormsByUserID :many
SELECT
    b.form_id,
    f.title,
    f.description,
    f.author_id,
    f.created_at
FROM bookmarks b
         JOIN forms f ON b.form_id = f.id
WHERE b.user_id = $1;
-- name: Create :one
INSERT INTO bookmarks (form_id, user_id)
VALUES ($1, $2)
    RETURNING *;

-- name: Delete :exec
DELETE FROM bookmarks
WHERE form_id = $1 AND user_id = $2;

-- name: Exist :one
SELECT EXISTS (SELECT 1 FROM bookmarks WHERE form_id = $1 AND user_id = $2) AS exists;

-- name: CountByUserID :one
SELECT COUNT(*) FROM bookmarks WHERE user_id = $1;

-- name: CountByFormID :one
SELECT COUNT(*) FROM bookmarks WHERE form_id = $1;