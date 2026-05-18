-- name: Create :one
INSERT INTO forms (title, description, author_id)
VALUES ($1, $2, $3)
    RETURNING *;

-- name: List :many
SELECT
    f.id,
    f.title,
    f.description,
    f.author_id,
    f.created_at,
    EXISTS(
        SELECT 1 FROM bookmarks b
        WHERE b.form_id = f.id AND b.user_id = $1
    ) AS is_bookmark
FROM forms f;

-- name: Update :one
UPDATE forms
SET title = $1, description = $2
WHERE id = $3
    RETURNING *;

-- name: Delete :exec
DELETE from forms where id =$1;

/*
-- name: 函式名稱 :回傳類型
在這裡寫sql然後透過sqlc產生go
*/