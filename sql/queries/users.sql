-- name: CreateUser :one 
INSERT INTO users (id, name) VALUES ($1, $2) RETURNING id, name, created_at, updated_at;

-- name: GetUser :one
SELECT id, name, created_at, updated_at FROM users WHERE name = $1;

-- name: GetAllUsers :many
SELECT id, name, created_at, updated_at FROM users;

-- name: ClearUsers :exec
DELETE FROM users;