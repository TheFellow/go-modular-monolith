package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	// Register the CGO-free SQLite database/sql driver.
	_ "modernc.org/sqlite"
)

// Store owns an embedded SQLite database. SQLite's WAL and busy timeout allow
// independent application processes on the same host to safely share it.
type Store struct{ db *sql.DB }

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Tx struct {
	tx    sqlExecutor
	sqlTx *sql.Tx
	conn  *sql.Conn
	ctx   context.Context
}

func Open(ctx context.Context, path string) (*Store, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, errors.Internalf("mkdir db dir: %w", err)
		}
	}
	dsn := "file:" + url.PathEscape(path) + "?" +
		"_pragma=busy_timeout(10000)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, errors.Internalf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	if err := enableWAL(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func enableWAL(ctx context.Context, db *sql.DB) error {
	deadline := time.Now().Add(10 * time.Second)
	delay := 10 * time.Millisecond
	for {
		var mode string
		err := db.QueryRowContext(ctx, "PRAGMA journal_mode=WAL").Scan(&mode)
		if err == nil {
			if !strings.EqualFold(mode, "wal") {
				return errors.Internalf("enable sqlite WAL: database selected %q", mode)
			}
			return nil
		}
		if !isBusy(err) || time.Now().After(deadline) {
			return errors.Internalf("enable sqlite WAL: %w", err)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Internalf("enable sqlite WAL: %w", ctx.Err())
		case <-timer.C:
		}
		delay = min(delay*2, 200*time.Millisecond)
	}
}

func (s *Store) migrate(ctx context.Context) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return errors.Internalf("acquire migration connection: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return errors.Internalf("begin migrations: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.WithoutCancel(ctx), "ROLLBACK")
		}
	}()
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
)`); err != nil {
		return errors.Internalf("create migration ledger: %w", err)
	}
	var current, applied int
	if err := conn.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0), COUNT(*) FROM schema_migrations").Scan(&current, &applied); err != nil {
		return errors.Internalf("read schema version: %w", err)
	}
	if current > len(schemaMigrations) {
		return errors.Internalf("database schema version %d is newer than supported version %d", current, len(schemaMigrations))
	}
	if applied != current {
		return errors.Internalf("invalid schema migration ledger: highest version %d with %d applied migrations", current, applied)
	}
	for i := current; i < len(schemaMigrations); i++ {
		version := i + 1
		if _, err := conn.ExecContext(ctx, schemaMigrations[i]); err != nil {
			return errors.Internalf("apply schema migration %d: %w", version, err)
		}
		if _, err := conn.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES (?, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))", version); err != nil {
			return errors.Internalf("record schema migration %d: %w", version, err)
		}
	}
	if _, err := conn.ExecContext(context.WithoutCancel(ctx), "COMMIT"); err != nil {
		return errors.Internalf("commit migrations: %w", err)
	}
	committed = true
	return nil
}

var schemaMigrations = []string{`
CREATE TABLE records (
  model TEXT NOT NULL,
  id TEXT NOT NULL,
  data TEXT NOT NULL CHECK(json_valid(data)),
  PRIMARY KEY(model, id)
)`}

// Register installs indexes declared by store tags on a domain's private row
// type. It is idempotent and safe when several processes start concurrently.
func (s *Store) Register(ctx context.Context, models ...any) {
	for _, model := range models {
		t := reflect.TypeOf(model)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		name := modelName(t)
		for f := range t.Fields() {
			tag := f.Tag.Get("store")
			if tag == "" {
				continue
			}
			unique := strings.Contains(tag, "unique")
			columns := []string{f.Name}
			if p := strings.Index(tag, "unique="); p >= 0 {
				columns = strings.Split(strings.TrimPrefix(tag[p:], "unique="), "+")
			}
			parts := make([]string, 0, len(columns))
			for _, c := range columns {
				expression := fmt.Sprintf("json_extract(data, '$.%s')", c)
				if indexedField, ok := t.FieldByName(c); ok && (indexedField.Type == reflect.TypeFor[time.Time]() || (indexedField.Type.Kind() == reflect.Pointer && indexedField.Type.Elem() == reflect.TypeFor[time.Time]())) {
					expression = "julianday(" + expression + ")"
				}
				parts = append(parts, expression)
			}
			kind := "INDEX"
			if unique {
				kind = "UNIQUE INDEX"
			}
			indexName := safeName("idx_" + name + "_" + strings.Join(columns, "_"))
			stmt := fmt.Sprintf("CREATE %s IF NOT EXISTS %s ON records(%s) WHERE model = %s", kind, indexName, strings.Join(parts, ", "), sqlStringLiteral(name))
			if _, err := s.db.ExecContext(ctx, stmt); err != nil {
				panic(err)
			}
		}
	}
}

func safeName(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func sqlStringLiteral(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func modelName(t reflect.Type) string { return t.PkgPath() + "." + t.Name() }

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Begin(ctx context.Context, writable bool) (*Tx, error) {
	var out *Tx
	if writable {
		conn, err := s.db.Conn(ctx)
		if err != nil {
			return nil, err
		}
		if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
			_ = conn.Close()
			return nil, err
		}
		out = &Tx{tx: conn, conn: conn, ctx: ctx}
	} else {
		tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			return nil, err
		}
		out = &Tx{tx: tx, sqlTx: tx, ctx: ctx}
	}
	registerTransaction(out)
	return out, nil
}

func (s *Store) Commit(tx *Tx) error {
	defer unregisterTransaction(tx)
	if tx.conn != nil {
		defer func() { _ = tx.conn.Close() }()
		_, err := tx.conn.ExecContext(context.WithoutCancel(tx.ctx), "COMMIT")
		return err
	}
	return tx.sqlTx.Commit()
}
func (s *Store) Rollback(tx *Tx) error {
	defer unregisterTransaction(tx)
	if tx.conn != nil {
		defer func() { _ = tx.conn.Close() }()
		_, err := tx.conn.ExecContext(context.WithoutCancel(tx.ctx), "ROLLBACK")
		return err
	}
	return tx.sqlTx.Rollback()
}

func (s *Store) Read(ctx context.Context, fn func(*Tx) error) error {
	start := time.Now()
	tx, err := s.Begin(ctx, false)
	if err == nil {
		err = fn(tx)
		if err == nil {
			err = s.Commit(tx)
		} else {
			_ = s.Rollback(tx)
		}
	}
	telemetry.FromContext(ctx).Histogram(telemetry.MetricStoreReadDuration).ObserveDuration(start)
	return err
}

func (s *Store) Write(ctx context.Context, fn func(*Tx) error) error {
	start := time.Now()
	tx, err := s.Begin(ctx, true)
	if err == nil {
		err = fn(tx)
		if err == nil {
			err = s.Commit(tx)
		} else {
			_ = s.Rollback(tx)
		}
	}
	telemetry.FromContext(ctx).Histogram(telemetry.MetricStoreWriteDuration).ObserveDuration(start)
	return err
}
