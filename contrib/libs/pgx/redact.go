package pgx

import (
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// RedactPgError is an xlog.Redactor for *pgconn.PgError: it renders only the
// SQLSTATE code and the constraint, table and column names. Message, Detail,
// Hint, Where and the query position are dropped: PostgreSQL puts row values
// there ("Key (email)=(a@b.c) already exists", `invalid input syntax for type
// integer: "abc"`).
//
//	render := xlog.NewErrorRenderer(xlog.RedactURL, pgx.RedactPgError)
func RedactPgError(err error) (string, bool) {
	switch e := err.(type) {
	case *pgconn.PgError:
		parts := []string{"SQLSTATE " + e.Code}
		for _, p := range [][2]string{
			{"constraint", e.ConstraintName},
			{"schema", e.SchemaName},
			{"table", e.TableName},
			{"column", e.ColumnName},
		} {
			if p[1] != "" {
				parts = append(parts, p[0]+" "+p[1])
			}
		}
		return strings.Join(parts, " "), true
	}
	return "", false
}
