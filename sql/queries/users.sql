-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
	gen_random_uuid(),
	NOW(),
	NOW(),
	$1,
	$2
)
RETURNING *;

-- name: GetUserByMail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserFromRefreshToken :one
SELECT u.id FROM users u INNER JOIN refresh_tokens r ON u.id = r.user_id WHERE r.token = $1 AND r.expires_at > NOW() AND r.revoked_at IS NULL;

-- name: UpdateUserPassAndMailByID :one
UPDATE users SET email=$2, hashed_password = $3, updated_at = NOW() WHERE id = $1 RETURNING *; 

-- name: DeleteAllUsers :exec
DELETE FROM users;
