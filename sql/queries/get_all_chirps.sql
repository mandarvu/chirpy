-- name: GetAllChirps :many
SELECT
    id,
    created_at,
    updated_at,
    body,
    user_id
FROM chirps
WHERE (user_id = sqlc.narg(user_id) OR sqlc.narg(user_id) IS NULL)
ORDER BY created_at ASC;
