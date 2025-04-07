-- name: SaveData :exec
INSERT INTO user_data (user_id, data_type, data_name, data_encrypted)
VALUES ($1, $2, $3, $4);

-- name: GetData :one
SELECT data_type, data_encrypted
FROM user_data
WHERE id = $1;

-- name: ListUserData :many
SELECT id, data_type, data_name
FROM user_data
WHERE user_id = $1;

-- name: CreateUser :one
INSERT INTO users (username, password_hash)
VALUES ($1, $2) RETURNING id;

-- name: IsUserCreated :one
SELECT EXISTS (SELECT 1
               FROM users
               WHERE username = $1);

-- name: CheckSessionUser :one
SELECT EXISTS (SELECT 1
               FROM user_sessions
               WHERE user_id = $1
                 AND session_token = $2);

-- name: GetUserCredentialsByUsername :one
SELECT id, password_hash
FROM users
WHERE username = $1;

-- name: DeleteUserData :exec
DELETE
FROM user_data
WHERE id = $1;

-- name: UpdateUserData :exec
UPDATE user_data
SET data_encrypted = $1
WHERE id = $2;

-- name: InsertUserSession :exec
INSERT INTO user_sessions (user_id, session_token, expires_at)
VALUES ($1, $2, NOW() + INTERVAL '20 minutes');
