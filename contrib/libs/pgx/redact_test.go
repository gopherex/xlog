package pgx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/gopherex/xlog"
)

func TestRedactPgError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
		want string
		ok   bool
	}{
		{name: "unique violation", err: &pgconn.PgError{Code: "23505", Message: `duplicate key value violates unique constraint "users_email_key"`,
			Detail: "Key (email)=(alice@example.com) already exists.", SchemaName: "public", TableName: "users",
			ConstraintName: "users_email_key"}, want: "SQLSTATE 23505 constraint users_email_key schema public table users", ok: true},
		{name: "value in the message", err: &pgconn.PgError{Code: "22P02",
			Message: `invalid input syntax for type integer: "secret"`, Where: "unnamed portal parameter $1 = '...'"},
			want: "SQLSTATE 22P02", ok: true},
		{name: "not a pg error", err: errors.New("x")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := RedactPgError(tc.err)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("RedactPgError = %q, %v; want %q, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

// Real PostgreSQL errors, wrapped as an application returns them, render
// without the values PostgreSQL put into them. Needs XLOG_TEST_PG.
func TestRedactRealPgErrors(t *testing.T) {
	dsn := os.Getenv("XLOG_TEST_PG")
	if dsn == "" {
		t.Skip("XLOG_TEST_PG not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `CREATE TEMP TABLE users(id int, email text CONSTRAINT users_email_key UNIQUE)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO users VALUES (1, 'alice@example.com')`); err != nil {
		t.Fatal(err)
	}
	_, dup := conn.Exec(ctx, `INSERT INTO users VALUES (2, 'alice@example.com')`)
	_, syntax := conn.Exec(ctx, `SELECT $1::int`, "secret-value")

	render := xlog.NewErrorRenderer(RedactPgError)
	for name, raw := range map[string]error{"unique violation": dup, "invalid input": syntax} {
		var pgErr *pgconn.PgError
		if !errors.As(raw, &pgErr) {
			t.Fatalf("%s: not a PgError: %v", name, raw)
		}
		got := render(fmt.Errorf("create user: %w", raw))
		if strings.Contains(got, "alice") || strings.Contains(got, "secret-value") || !strings.HasPrefix(got, "create user: SQLSTATE "+pgErr.Code) {
			t.Fatalf("%s rendered %q", name, got)
		}
		t.Logf("%s: %q", name, got)
	}
}
