-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, expires_at, revoked_at, user_id)
VALUES (
	$1,
	NOW(),
	NOW(),
	$2,
    NULL,
    $3
)
RETURNING *;

-- name: RevokeTokenAccess :exec
UPDATE refresh_tokens SET expires_at = $2 WHERE token = $1;


-- name: DeleteAllRefreshTokens :exec
DELETE FROM refresh_tokens;
