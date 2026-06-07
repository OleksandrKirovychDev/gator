-- name: CreateFeed :one
INSERT INTO feed (id, name, url, user_id) VALUES ($1, $2, $3, $4) RETURNING id, name, url, user_id, created_at, updated_at;

-- name: GetAllFeeds :many
SELECT feed.id , feed.name, feed.url, users.name AS user_name FROM feed JOIN users ON feed.user_id = users.id;

-- name: GetFeedByURL :one
SELECT feed.id , feed.name, feed.url, users.name AS user_name FROM feed JOIN users ON feed.user_id = users.id WHERE feed.url = $1;

-- name: MarkFeedFetched :exec
UPDATE feed
SET last_fetched_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: GetNextFeedToFetch :one
SELECT id, name, url, user_id, created_at, updated_at, last_fetched_at
FROM feed
ORDER BY last_fetched_at ASC NULLS FIRST
LIMIT 1;