-- name: ChirpAdd :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(), NOW(), NOW(), $1, $2
)
RETURNING *;

-- name: ChirpGet :one
SELECT * FROM chirps
WHERE id = $1 LIMIT 1;

-- name: ChirpsGet :many
SELECT * FROM chirps
ORDER BY created_at ASC;
