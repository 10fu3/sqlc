package main

import (
	"context"
	"database/sql"
	"testing"

	db "github.com/sqlc-dev/sqlc/endtoend/emit_embed_nullable_left_join/db"
)

func TestNullableEmbed(t *testing.T) {
	// This is just a compile test to ensure the generated code compiles correctly
	// We're not actually running it against a database
	
	// Test that we can reference the generated types and methods
	var _ db.ListPostsWithAuthorsRow
	var _ sql.Null[db.Author]
	
	// Test that we can compile code that uses the nullable embed
	compileTest := func() {
		ctx := context.Background()
		var dbConn *sql.DB
		q := db.New(dbConn)
		
		// This would panic at runtime but compiles fine
		_ = func() ([]db.ListPostsWithAuthorsRow, error) {
			return q.ListPostsWithAuthors(ctx)
		}
		
		// Test that we can access fields properly
		var row db.ListPostsWithAuthorsRow
		if row.Author.Valid {
			// Author exists
			_ = row.Author.V.ID
			_ = row.Author.V.Name
			_ = row.Author.V.Bio
		}
	}
	_ = compileTest
}