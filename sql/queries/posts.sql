-- name: CreatePost :exec
INSERT INTO posts (id, title, url, description, published_at, feed_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetPostsForUser :many
SELECT
    posts.id,
    posts.created_at,
    posts.updated_at,
    posts.title,
    posts.url,
    posts.description,
    posts.published_at,
    posts.feed_id,
    feed.name AS feed_name
FROM posts
JOIN feed ON posts.feed_id = feed.id
JOIN feed_follows ON posts.feed_id = feed_follows.feed_id
WHERE feed_follows.user_id = $1
ORDER BY posts.published_at DESC NULLS LAST
LIMIT $2;
