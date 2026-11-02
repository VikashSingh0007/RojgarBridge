package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type DB struct {
	conn       *sql.DB
	driverName string // "postgres" or "sqlite"
	mu         sync.Mutex
}

func (d *DB) Conn() *sql.DB {
	return d.conn
}

func (d *DB) DriverName() string {
	return d.driverName
}

func (d *DB) Lock() {
	d.mu.Lock()
}

func (d *DB) Unlock() {
	d.mu.Unlock()
}

func (d *DB) Rebind(query string) string {
	if d.driverName != "postgres" {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 20)
	argIdx := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			b.WriteString("$")
			b.WriteString(strconv.Itoa(argIdx))
			argIdx++
		} else {
			b.WriteByte(query[i])
		}
	}
	return b.String()
}

func (d *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.conn.Query(d.Rebind(query), args...)
}

func (d *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.conn.QueryRow(d.Rebind(query), args...)
}

func (d *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return d.conn.Exec(d.Rebind(query), args...)
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func InitDB(dbPath, databaseURL string) (*DB, error) {
	if databaseURL != "" {
		// Set pure Go public DNS resolver fallback to prevent local router SERVFAIL
		net.DefaultResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, "udp", "8.8.8.8:53")
			},
		}

		conn, err := sql.Open("postgres", databaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to open postgres database: %w", err)
		}

		conn.SetMaxOpenConns(50)
		conn.SetMaxIdleConns(25)
		conn.SetConnMaxLifetime(15 * time.Minute)

		if err := conn.Ping(); err != nil {
			return nil, fmt.Errorf("failed to ping postgres database: %w", err)
		}

		db := &DB{conn: conn, driverName: "postgres"}
		if err := db.migrate(); err != nil {
			return nil, fmt.Errorf("postgres migration failed: %w", err)
		}

		return db, nil
	}

	// Fallback to SQLite
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Optimize SQLite performance & concurrency (WAL, 64MB Cache, 256MB MMAP)
	if _, err := conn.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
		PRAGMA busy_timeout = 15000;
		PRAGMA cache_size = -64000;
		PRAGMA mmap_size = 268435456;
		PRAGMA temp_store = MEMORY;
	`); err != nil {
		return nil, fmt.Errorf("failed to set pragma: %w", err)
	}

	conn.SetMaxOpenConns(50)
	conn.SetMaxIdleConns(25)

	db := &DB{conn: conn, driverName: "sqlite"}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("sqlite migration failed: %w", err)
	}

	return db, nil
}

func (d *DB) migrate() error {
	if d.driverName == "postgres" {
		query := `
		CREATE OR REPLACE FUNCTION json_extract(j jsonb, path text) RETURNS text AS $$
		BEGIN
			RETURN j ->> substring(path from 3);
		END;
		$$ LANGUAGE plpgsql IMMUTABLE;

		CREATE OR REPLACE FUNCTION safe_cast_int(val text) RETURNS integer AS $$
		BEGIN
			RETURN COALESCE(val::integer, 0);
		EXCEPTION WHEN OTHERS THEN
			RETURN 0;
		END;
		$$ LANGUAGE plpgsql IMMUTABLE;

		CREATE TABLE IF NOT EXISTS jobs (
			id SERIAL PRIMARY KEY,
			hash TEXT UNIQUE NOT NULL,
			source_portal TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'draft',
			title TEXT NOT NULL DEFAULT '',
			notice_type TEXT NOT NULL DEFAULT 'recruitment',
			organization TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT '',
			state TEXT NOT NULL DEFAULT 'All India',
			advt_number TEXT NOT NULL DEFAULT '',
			total_vacancies INTEGER NOT NULL DEFAULT 0,
			min_age INTEGER,
			max_age INTEGER,
			qualification TEXT NOT NULL DEFAULT '',
			application_start TEXT NOT NULL DEFAULT '',
			application_end TEXT NOT NULL DEFAULT '',
			exam_date TEXT NOT NULL DEFAULT '',
			fee_general TEXT NOT NULL DEFAULT '',
			fee_reserved TEXT NOT NULL DEFAULT '',
			salary_payscale TEXT NOT NULL DEFAULT '',
			official_apply_url TEXT NOT NULL DEFAULT '',
			official_notice_url TEXT NOT NULL DEFAULT '',
			confidence_score REAL NOT NULL DEFAULT 1.0,
			raw_notice JSONB NOT NULL,
			details JSONB NOT NULL,
			review_notes TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			published_at TIMESTAMPTZ
		);

		CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
		CREATE INDEX IF NOT EXISTS idx_jobs_hash ON jobs(hash);
		CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at DESC);

		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT 'user',
			phone TEXT DEFAULT '',
			google_id TEXT DEFAULT '',
			avatar_url TEXT DEFAULT '',
			auth_provider TEXT DEFAULT 'local',
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
		`
		if _, err := d.conn.Exec(query); err != nil {
			return err
		}
		d.ensureJobColumns()
		d.ensureUserColumns()
		return nil
	}

	query := `
	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hash TEXT UNIQUE NOT NULL,
		source_portal TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'draft',
		title TEXT NOT NULL DEFAULT '',
		notice_type TEXT NOT NULL DEFAULT 'recruitment',
		organization TEXT NOT NULL DEFAULT '',
		category TEXT NOT NULL DEFAULT '',
		state TEXT NOT NULL DEFAULT 'All India',
		advt_number TEXT NOT NULL DEFAULT '',
		total_vacancies INTEGER NOT NULL DEFAULT 0,
		min_age INTEGER,
		max_age INTEGER,
		qualification TEXT NOT NULL DEFAULT '',
		application_start TEXT NOT NULL DEFAULT '',
		application_end TEXT NOT NULL DEFAULT '',
		exam_date TEXT NOT NULL DEFAULT '',
		fee_general TEXT NOT NULL DEFAULT '',
		fee_reserved TEXT NOT NULL DEFAULT '',
		salary_payscale TEXT NOT NULL DEFAULT '',
		official_apply_url TEXT NOT NULL DEFAULT '',
		official_notice_url TEXT NOT NULL DEFAULT '',
		confidence_score REAL NOT NULL DEFAULT 1.0,
		raw_notice JSON NOT NULL,
		details JSON NOT NULL,
		review_notes TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		published_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
	CREATE INDEX IF NOT EXISTS idx_jobs_hash ON jobs(hash);
	CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at DESC);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL DEFAULT 'user',
		phone TEXT DEFAULT '',
		google_id TEXT DEFAULT '',
		avatar_url TEXT DEFAULT '',
		auth_provider TEXT DEFAULT 'local',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`
	if _, err := d.conn.Exec(query); err != nil {
		return err
	}
	d.ensureJobColumns()
	d.ensureUserColumns()
	return nil
}

func (d *DB) ensureJobColumns() {
	if d.driverName == "postgres" {
		_, _ = d.conn.Exec(`
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS notice_type TEXT NOT NULL DEFAULT 'recruitment';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS organization TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS state TEXT NOT NULL DEFAULT 'All India';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS advt_number TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS total_vacancies INTEGER NOT NULL DEFAULT 0;
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS min_age INTEGER;
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS max_age INTEGER;
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS qualification TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS application_start TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS application_end TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS exam_date TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS fee_general TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS fee_reserved TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS salary_payscale TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS official_apply_url TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS official_notice_url TEXT NOT NULL DEFAULT '';
			ALTER TABLE jobs ADD COLUMN IF NOT EXISTS confidence_score REAL NOT NULL DEFAULT 1.0;

			UPDATE jobs SET
				title = COALESCE(NULLIF(title, ''), details->>'title', ''),
				notice_type = COALESCE(NULLIF(details->>'notice_type', ''), NULLIF(notice_type, ''), 'recruitment'),
				organization = COALESCE(NULLIF(organization, ''), details->>'organization', ''),
				category = COALESCE(NULLIF(category, ''), details->>'category', ''),
				state = COALESCE(NULLIF(state, ''), NULLIF(details->>'state', ''), 'All India'),
				advt_number = COALESCE(NULLIF(advt_number, ''), details->>'advt_number', ''),
				total_vacancies = CASE WHEN total_vacancies > 0 THEN total_vacancies ELSE COALESCE(safe_cast_int(details->>'total_vacancies'), 0) END,
				qualification = COALESCE(NULLIF(qualification, ''), details->>'qualification', ''),
				application_start = COALESCE(NULLIF(application_start, ''), details->>'application_start', ''),
				application_end = COALESCE(NULLIF(application_end, ''), details->>'application_end', ''),
				exam_date = COALESCE(NULLIF(exam_date, ''), details->>'exam_date', ''),
				fee_general = COALESCE(NULLIF(fee_general, ''), details->>'fee_general', ''),
				fee_reserved = COALESCE(NULLIF(fee_reserved, ''), details->>'fee_reserved', ''),
				salary_payscale = COALESCE(NULLIF(salary_payscale, ''), details->>'salary_payscale', ''),
				official_apply_url = COALESCE(NULLIF(official_apply_url, ''), details->>'official_apply_url', ''),
				official_notice_url = COALESCE(NULLIF(official_notice_url, ''), details->>'official_notice_url', '')
			WHERE title = '' OR title IS NULL;

			UPDATE jobs SET notice_type = details->>'notice_type'
			WHERE details->>'notice_type' IS NOT NULL AND details->>'notice_type' != '' AND (notice_type IS NULL OR notice_type != details->>'notice_type');

			CREATE INDEX IF NOT EXISTS idx_jobs_status_pub ON jobs(status, published_at DESC, created_at DESC);
			CREATE INDEX IF NOT EXISTS idx_jobs_notice_type ON jobs(notice_type);
			CREATE INDEX IF NOT EXISTS idx_jobs_category ON jobs(category);
			CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs(state);
			CREATE INDEX IF NOT EXISTS idx_jobs_org ON jobs(organization);
			CREATE INDEX IF NOT EXISTS idx_jobs_vacancies ON jobs(total_vacancies DESC);
			CREATE INDEX IF NOT EXISTS idx_jobs_app_end ON jobs(application_end);
			CREATE INDEX IF NOT EXISTS idx_jobs_portal ON jobs(source_portal);
		`)
		return
	}

	rows, err := d.conn.Query("PRAGMA table_info(jobs)")
	if err != nil {
		return
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err == nil {
			existing[name] = true
		}
	}

	cols := []struct {
		name string
		def  string
	}{
		{"title", "TEXT NOT NULL DEFAULT ''"},
		{"notice_type", "TEXT NOT NULL DEFAULT 'recruitment'"},
		{"organization", "TEXT NOT NULL DEFAULT ''"},
		{"category", "TEXT NOT NULL DEFAULT ''"},
		{"state", "TEXT NOT NULL DEFAULT 'All India'"},
		{"advt_number", "TEXT NOT NULL DEFAULT ''"},
		{"total_vacancies", "INTEGER NOT NULL DEFAULT 0"},
		{"min_age", "INTEGER"},
		{"max_age", "INTEGER"},
		{"qualification", "TEXT NOT NULL DEFAULT ''"},
		{"application_start", "TEXT NOT NULL DEFAULT ''"},
		{"application_end", "TEXT NOT NULL DEFAULT ''"},
		{"exam_date", "TEXT NOT NULL DEFAULT ''"},
		{"fee_general", "TEXT NOT NULL DEFAULT ''"},
		{"fee_reserved", "TEXT NOT NULL DEFAULT ''"},
		{"salary_payscale", "TEXT NOT NULL DEFAULT ''"},
		{"official_apply_url", "TEXT NOT NULL DEFAULT ''"},
		{"official_notice_url", "TEXT NOT NULL DEFAULT ''"},
		{"confidence_score", "REAL NOT NULL DEFAULT 1.0"},
	}

	for _, col := range cols {
		if !existing[col.name] {
			_, _ = d.conn.Exec(fmt.Sprintf("ALTER TABLE jobs ADD COLUMN %s %s", col.name, col.def))
		}
	}

	// Backfill and index SQLite
	_, _ = d.conn.Exec(`
		UPDATE jobs SET
			title = COALESCE(NULLIF(title, ''), json_extract(details, '$.title'), ''),
			notice_type = COALESCE(NULLIF(json_extract(details, '$.notice_type'), ''), NULLIF(notice_type, ''), 'recruitment'),
			organization = COALESCE(NULLIF(organization, ''), json_extract(details, '$.organization'), ''),
			category = COALESCE(NULLIF(category, ''), json_extract(details, '$.category'), ''),
			state = COALESCE(NULLIF(state, ''), NULLIF(json_extract(details, '$.state'), ''), 'All India'),
			advt_number = COALESCE(NULLIF(advt_number, ''), json_extract(details, '$.advt_number'), ''),
			total_vacancies = CASE WHEN total_vacancies > 0 THEN total_vacancies ELSE COALESCE(CAST(json_extract(details, '$.total_vacancies') AS INTEGER), 0) END,
			qualification = COALESCE(NULLIF(qualification, ''), json_extract(details, '$.qualification'), ''),
			application_start = COALESCE(NULLIF(application_start, ''), json_extract(details, '$.application_start'), ''),
			application_end = COALESCE(NULLIF(application_end, ''), json_extract(details, '$.application_end'), ''),
			exam_date = COALESCE(NULLIF(exam_date, ''), json_extract(details, '$.exam_date'), ''),
			fee_general = COALESCE(NULLIF(fee_general, ''), json_extract(details, '$.fee_general'), ''),
			fee_reserved = COALESCE(NULLIF(fee_reserved, ''), json_extract(details, '$.fee_reserved'), ''),
			salary_payscale = COALESCE(NULLIF(salary_payscale, ''), json_extract(details, '$.salary_payscale'), ''),
			official_apply_url = COALESCE(NULLIF(official_apply_url, ''), json_extract(details, '$.official_apply_url'), ''),
			official_notice_url = COALESCE(NULLIF(official_notice_url, ''), json_extract(details, '$.official_notice_url'), '')
		WHERE title = '' OR title IS NULL;

		CREATE INDEX IF NOT EXISTS idx_jobs_status_pub ON jobs(status, published_at DESC, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_jobs_notice_type ON jobs(notice_type);
		CREATE INDEX IF NOT EXISTS idx_jobs_category ON jobs(category);
		CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs(state);
		CREATE INDEX IF NOT EXISTS idx_jobs_org ON jobs(organization);
		CREATE INDEX IF NOT EXISTS idx_jobs_vacancies ON jobs(total_vacancies DESC);
		CREATE INDEX IF NOT EXISTS idx_jobs_app_end ON jobs(application_end);
		CREATE INDEX IF NOT EXISTS idx_jobs_portal ON jobs(source_portal);
	`)
}

func (d *DB) ensureUserColumns() {
	if d.driverName == "postgres" {
		_, _ = d.conn.Exec(`
			ALTER TABLE users ADD COLUMN IF NOT EXISTS google_id TEXT DEFAULT '';
			ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT DEFAULT '';
			ALTER TABLE users ADD COLUMN IF NOT EXISTS auth_provider TEXT DEFAULT 'local';
			ALTER TABLE users ADD COLUMN IF NOT EXISTS phone TEXT DEFAULT '';
			CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);
		`)
		return
	}

	rows, err := d.conn.Query("PRAGMA table_info(users)")
	if err != nil {
		return
	}
	defer rows.Close()

	hasGoogleID := false
	hasAvatarURL := false
	hasAuthProvider := false
	hasPhone := false

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err == nil {
			switch name {
			case "google_id":
				hasGoogleID = true
			case "avatar_url":
				hasAvatarURL = true
			case "auth_provider":
				hasAuthProvider = true
			case "phone":
				hasPhone = true
			}
		}
	}

	if !hasGoogleID {
		_, _ = d.conn.Exec("ALTER TABLE users ADD COLUMN google_id TEXT DEFAULT ''")
	}
	if !hasAvatarURL {
		_, _ = d.conn.Exec("ALTER TABLE users ADD COLUMN avatar_url TEXT DEFAULT ''")
	}
	if !hasAuthProvider {
		_, _ = d.conn.Exec("ALTER TABLE users ADD COLUMN auth_provider TEXT DEFAULT 'local'")
	}
	if !hasPhone {
		_, _ = d.conn.Exec("ALTER TABLE users ADD COLUMN phone TEXT DEFAULT ''")
	}

	_, _ = d.conn.Exec("CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id)")
}
