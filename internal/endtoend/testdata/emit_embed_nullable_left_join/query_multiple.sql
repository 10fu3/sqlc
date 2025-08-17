-- name: ListPostsWithAuthorAndReviewer :many
SELECT 
  p.id, 
  p.title, 
  p.body, 
  p.author_id,
  p.reviewer_id,
  sqlc.embed(a1),
  sqlc.embed(a2)
FROM posts p
LEFT JOIN authors a1 ON a1.id = p.author_id
LEFT JOIN authors a2 ON a2.id = p.reviewer_id;