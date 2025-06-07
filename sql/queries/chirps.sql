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
WHERE (COALESCE($1::uuid, '00000000-0000-0000-0000-000000000000') = '00000000-0000-0000-0000-000000000000' OR user_id = $1)
ORDER BY created_at ASC;

-- name: ChirpDelete :exec
DELETE FROM chirps
WHERE id = $1;
