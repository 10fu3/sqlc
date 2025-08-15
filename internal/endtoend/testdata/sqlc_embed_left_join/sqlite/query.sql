-- name: UserWithProfile :one
SELECT sqlc.embed(users), sqlc.embed(profiles)
FROM users
LEFT JOIN profiles ON users.id = profiles.user_id
WHERE users.id = ?;

-- name: UserWithPostsAndComments :one
SELECT sqlc.embed(users), sqlc.embed(posts), sqlc.embed(comments)
FROM users
INNER JOIN posts ON users.id = posts.user_id
LEFT JOIN comments ON posts.id = comments.post_id
WHERE users.id = ?;

-- name: ProfileWithUser :one
SELECT sqlc.embed(profiles), sqlc.embed(users)
FROM profiles
RIGHT JOIN users ON profiles.user_id = users.id
WHERE profiles.id = ?;

-- name: InnerJoinBaseline :one
SELECT sqlc.embed(users), sqlc.embed(posts)
FROM users
INNER JOIN posts ON users.id = posts.user_id
WHERE users.id = ?;

-- name: MultipleLeftJoins :one
SELECT sqlc.embed(users), sqlc.embed(profiles), sqlc.embed(posts)
FROM users
LEFT JOIN profiles ON users.id = profiles.user_id
LEFT JOIN posts ON users.id = posts.user_id
WHERE users.id = ?;