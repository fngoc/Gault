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

-- name: InsertUserSession :exec
INSERT INTO user_sessions (user_id, session_token, expires_at)
VALUES ($1, $2, NOW() + INTERVAL '20 minutes');

-- name: GetDataInfoByID :one
SELECT data_type, data_name, largeobject_oid
FROM user_data
WHERE id = $1;

-- name: InsertUserDataWithOid :exec
INSERT INTO user_data (id, user_id, data_type, data_name, largeobject_oid)
VALUES ($1, $2, $3, $4, $5);

-- name: GetOidByID :one
SELECT largeobject_oid
FROM user_data
WHERE id = $1;