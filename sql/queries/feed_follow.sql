-- name: CreateFeedFollow :one
WITH inserted AS (
    INSERT INTO feed_follows (id, user_id, feed_id)
    VALUES ($1, $2, $3)
    RETURNING feed_id
)
SELECT
    feed.id,
    feed.name,
    feed.url,
    users.name AS user_name
FROM inserted
JOIN feed ON feed.id = inserted.feed_id
JOIN users ON feed.user_id = users.id;

-- name: GetFollowsForUser :many
SELECT
    feed.id,
    feed.name,
    feed.url,
    users.name AS user_name
FROM feed_follows
JOIN feed ON feed_follows.feed_id = feed.id
JOIN users ON feed.user_id = users.id
WHERE feed_follows.user_id = $1;

-- name: UnfollowFeed :exec
DELETE FROM feed_follows WHERE user_id = $1 AND feed_id = $2;