-- name: ListPostsWithAuthors :many
SELECT p.*, sqlc.embed(authors)
FROM posts p
LEFT JOIN authors ON authors.id = p.author_id;