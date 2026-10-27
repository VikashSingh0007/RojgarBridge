package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
	"rojgarbridge-backend/internal/models"
)

type DB struct {
	conn       *sql.DB
	driverName string // "postgres" or "sqlite"
	mu         sync.Mutex
}

func (d *DB) DriverName() string {
	return d.driverName
}

func (d *DB) rebind(query string) string {
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

func InitDB(dbPath, databaseURL string) (*DB, error) {
	if databaseURL != "" {
		conn, err := sql.Open("postgres", databaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to open postgres database: %w", err)
		}

		conn.SetMaxOpenConns(25)
		conn.SetMaxIdleConns(10)
		conn.SetConnMaxLifetime(5 * time.Minute)

		if err := conn.Ping(); err != nil {
			return nil, fmt.Errorf("failed to ping postgres database: %w", err)
		}

		db := &DB{conn: conn, driverName: "postgres"}
		if err := db.migrate(); err != nil {
			return nil, fmt.Errorf("postgres migration failed: %w", err)
		}

		if err := db.EnsurePublishedJobs(); err != nil {
			return nil, fmt.Errorf("failed ensuring published jobs: %w", err)
		}

		if err := db.EnsureDefaultAdmin(); err != nil {
			return nil, fmt.Errorf("failed ensuring default admin user: %w", err)
		}

		return db, nil
	}

	// Fallback to SQLite
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Optimize SQLite performance & concurrency
	if _, err := conn.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
		PRAGMA busy_timeout = 10000;
	`); err != nil {
		return nil, fmt.Errorf("failed to set pragma: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(10)

	db := &DB{conn: conn, driverName: "sqlite"}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("sqlite migration failed: %w", err)
	}

	if err := db.EnsurePublishedJobs(); err != nil {
		return nil, fmt.Errorf("failed ensuring published jobs: %w", err)
	}

	if err := db.EnsureDefaultAdmin(); err != nil {
		return nil, fmt.Errorf("failed ensuring default admin user: %w", err)
	}

	return db, nil
}

// EnsurePublishedJobs ensures that in production or local, there are always published jobs available
func (d *DB) EnsurePublishedJobs() error {
	var pubCount int
	_ = d.conn.QueryRow("SELECT COUNT(1) FROM jobs WHERE status = 'published'").Scan(&pubCount)
	if pubCount == 0 {
		var draftCount int
		_ = d.conn.QueryRow("SELECT COUNT(1) FROM jobs WHERE status = 'draft'").Scan(&draftCount)
		if draftCount > 0 {
			// Auto-publish existing draft notifications so production is instantly active
			_, err := d.conn.Exec("UPDATE jobs SET status = 'published', published_at = CURRENT_TIMESTAMP WHERE status = 'draft'")
			if err != nil {
				return err
			}
		} else {
			// Database is completely empty, seed initial verified government notices
			return d.SeedInitialVerifiedJobs()
		}
	}
	return nil
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
		CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);
		`
		if _, err := d.conn.Exec(query); err != nil {
			return err
		}
		d.ensureUserColumns()
		return nil
	}

	query := `
	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hash TEXT UNIQUE NOT NULL,
		source_portal TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'draft',
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
	CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);
	`
	if _, err := d.conn.Exec(query); err != nil {
		return err
	}
	d.ensureUserColumns()
	return nil
}

func (d *DB) ensureUserColumns() {
	if d.driverName == "postgres" {
		_, _ = d.conn.Exec(`
			ALTER TABLE users ADD COLUMN IF NOT EXISTS google_id TEXT DEFAULT '';
			ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT DEFAULT '';
			ALTER TABLE users ADD COLUMN IF NOT EXISTS auth_provider TEXT DEFAULT 'local';
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
}

func (d *DB) ExistsByHash(hash string) (bool, error) {
	var count int
	err := d.conn.QueryRow(d.rebind("SELECT COUNT(1) FROM jobs WHERE hash = ?"), hash).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (d *DB) SaveDraft(job *models.Job) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	rawJSON, err := json.Marshal(job.RawNotice)
	if err != nil {
		return fmt.Errorf("failed to marshal raw notice: %w", err)
	}

	detailsJSON, err := json.Marshal(job.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	now := time.Now()
	if d.driverName == "postgres" {
		query := `
		INSERT INTO jobs (hash, source_portal, status, raw_notice, details, review_notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6, $7, $8)
		RETURNING id
		`
		var id int64
		err := d.conn.QueryRow(query, job.Hash, job.SourcePortal, models.StatusDraft, string(rawJSON), string(detailsJSON), job.ReviewNotes, now, now).Scan(&id)
		if err != nil {
			return fmt.Errorf("failed to insert draft: %w", err)
		}
		job.ID = id
		job.Status = models.StatusDraft
		job.CreatedAt = now
		job.UpdatedAt = now
		return nil
	}

	query := `
	INSERT INTO jobs (hash, source_portal, status, raw_notice, details, review_notes, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := d.conn.Exec(query, job.Hash, job.SourcePortal, models.StatusDraft, string(rawJSON), string(detailsJSON), job.ReviewNotes, now, now)
	if err != nil {
		return fmt.Errorf("failed to insert draft: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		job.ID = id
		job.Status = models.StatusDraft
		job.CreatedAt = now
		job.UpdatedAt = now
	}
	return nil
}

type JobFilterParams struct {
	Status         string
	Query          string
	NoticeTypes    []string
	Categories     []string
	Qualifications []string
	States         []string
	SortBy         string
	Limit          int
	Offset         int
}

func (d *DB) QueryJobs(params JobFilterParams) ([]models.Job, int, error) {
	var whereClauses []string
	var args []interface{}

	// 1. Status (published, draft, etc.)
	if params.Status != "" && params.Status != "all" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, params.Status)
	}

	// 2. Search query across all relevant fields
	if strings.TrimSpace(params.Query) != "" {
		term := "%" + strings.TrimSpace(params.Query) + "%"
		if d.driverName == "postgres" {
			whereClauses = append(whereClauses, "(json_extract(details, '$.title') ILIKE ? OR json_extract(details, '$.organization') ILIKE ? OR json_extract(details, '$.advt_number') ILIKE ? OR json_extract(details, '$.qualification') ILIKE ? OR json_extract(details, '$.category') ILIKE ? OR json_extract(details, '$.state') ILIKE ? OR source_portal ILIKE ? OR raw_notice::text ILIKE ?)")
		} else {
			whereClauses = append(whereClauses, "(json_extract(details, '$.title') LIKE ? OR json_extract(details, '$.organization') LIKE ? OR json_extract(details, '$.advt_number') LIKE ? OR json_extract(details, '$.qualification') LIKE ? OR json_extract(details, '$.category') LIKE ? OR json_extract(details, '$.state') LIKE ? OR source_portal LIKE ? OR raw_notice LIKE ?)")
		}
		args = append(args, term, term, term, term, term, term, term, term)
	}

	// 3. Notice types filter
	if len(params.NoticeTypes) > 0 {
		var typeOrs []string
		for _, t := range params.NoticeTypes {
			tClean := strings.TrimSpace(strings.ToLower(t))
			if tClean == "" || tClean == "all" {
				continue
			}
			if strings.Contains(tClean, "recruitment") {
				typeOrs = append(typeOrs, "(json_extract(details, '$.notice_type') = 'recruitment' OR json_extract(details, '$.notice_type') IS NULL OR json_extract(details, '$.notice_type') = '')")
			} else if strings.Contains(tClean, "admit") {
				typeOrs = append(typeOrs, "json_extract(details, '$.notice_type') = 'admit_card'")
			} else if strings.Contains(tClean, "result") {
				typeOrs = append(typeOrs, "json_extract(details, '$.notice_type') = 'result'")
			} else if strings.Contains(tClean, "answer") {
				typeOrs = append(typeOrs, "json_extract(details, '$.notice_type') = 'answer_key'")
			} else {
				typeOrs = append(typeOrs, "json_extract(details, '$.notice_type') = ?")
				args = append(args, tClean)
			}
		}
		if len(typeOrs) > 0 {
			whereClauses = append(whereClauses, "("+strings.Join(typeOrs, " OR ")+")")
		}
	}

	// 4. Categories & Department sectors
	if len(params.Categories) > 0 {
		var catOrs []string
		for _, cat := range params.Categories {
			cat = strings.TrimSpace(cat)
			if cat == "" || cat == "all" {
				continue
			}
			cLower := strings.ToLower(cat)
			if cLower == "central govt" {
				catOrs = append(catOrs, "(json_extract(details, '$.category') LIKE '%Central%' OR source_portal IN ('UPSC', 'SSC', 'RRB', 'IBPS', 'ISRO'))")
			} else if cLower == "state govt" {
				catOrs = append(catOrs, "(json_extract(details, '$.category') LIKE '%State%' OR source_portal IN ('UPPSC', 'BPSC', 'MPPSC', 'RPSC'))")
			} else if cLower == "banking" || cLower == "bank" {
				catOrs = append(catOrs, "(json_extract(details, '$.category') LIKE '%Bank%' OR json_extract(details, '$.organization') LIKE '%Bank%' OR source_portal = 'IBPS')")
			} else if strings.Contains(cLower, "defence") || strings.Contains(cLower, "police") {
				catOrs = append(catOrs, "(json_extract(details, '$.category') LIKE '%Defence%' OR json_extract(details, '$.category') LIKE '%Police%' OR json_extract(details, '$.title') LIKE '%Police%' OR json_extract(details, '$.title') LIKE '%Defence%')")
			} else if strings.Contains(cLower, "teach") {
				catOrs = append(catOrs, "(json_extract(details, '$.category') LIKE '%Teach%' OR json_extract(details, '$.title') LIKE '%Teacher%' OR json_extract(details, '$.title') LIKE '%Lecturer%' OR json_extract(details, '$.title') LIKE '%Principal%')")
			} else if strings.Contains(cLower, "engineering") || strings.Contains(cLower, "psu") {
				catOrs = append(catOrs, "(json_extract(details, '$.category') LIKE '%Engineering%' OR json_extract(details, '$.category') LIKE '%PSU%' OR json_extract(details, '$.title') LIKE '%Engineer%' OR source_portal = 'ISRO')")
			} else if cLower == "railway" {
				catOrs = append(catOrs, "(source_portal = 'RRB' OR json_extract(details, '$.organization') LIKE '%Railway%')")
			} else if cLower == "ssc" {
				catOrs = append(catOrs, "(source_portal = 'SSC' OR json_extract(details, '$.organization') LIKE '%Staff Selection%')")
			} else if cLower == "upsc" {
				catOrs = append(catOrs, "(source_portal = 'UPSC' OR json_extract(details, '$.organization') LIKE '%UPSC%')")
			} else {
				catOrs = append(catOrs, "(json_extract(details, '$.category') LIKE ? OR json_extract(details, '$.organization') LIKE ? OR source_portal = ?)")
				args = append(args, "%"+cat+"%", "%"+cat+"%", cat)
			}
		}
		if len(catOrs) > 0 {
			whereClauses = append(whereClauses, "("+strings.Join(catOrs, " OR ")+")")
		}
	}

	// 5. Qualification filters
	if len(params.Qualifications) > 0 {
		var qualOrs []string
		for _, q := range params.Qualifications {
			qClean := strings.TrimSpace(q)
			if qClean == "" {
				continue
			}
			qLower := strings.ToLower(qClean)
			if strings.Contains(qLower, "10th") {
				qualOrs = append(qualOrs, "(json_extract(details, '$.qualification') LIKE '%10th%' OR json_extract(details, '$.qualification') LIKE '%Matric%')")
			} else if strings.Contains(qLower, "12th") {
				qualOrs = append(qualOrs, "(json_extract(details, '$.qualification') LIKE '%12th%' OR json_extract(details, '$.qualification') LIKE '%Intermediate%' OR json_extract(details, '$.qualification') LIKE '%10+2%')")
			} else if strings.Contains(qLower, "iti") {
				qualOrs = append(qualOrs, "json_extract(details, '$.qualification') LIKE '%ITI%'")
			} else if strings.Contains(qLower, "diploma") {
				qualOrs = append(qualOrs, "json_extract(details, '$.qualification') LIKE '%Diploma%'")
			} else if strings.Contains(qLower, "b.tech") || strings.Contains(qLower, "b.e") {
				qualOrs = append(qualOrs, "(json_extract(details, '$.qualification') LIKE '%B.E%' OR json_extract(details, '$.qualification') LIKE '%B.Tech%' OR json_extract(details, '$.qualification') LIKE '%Engineer%')")
			} else if strings.Contains(qLower, "post graduate") || strings.Contains(qLower, "pg") {
				qualOrs = append(qualOrs, "(json_extract(details, '$.qualification') LIKE '%Post Graduate%' OR json_extract(details, '$.qualification') LIKE '%PG%' OR json_extract(details, '$.qualification') LIKE '%Master%')")
			} else if strings.Contains(qLower, "graduate") {
				qualOrs = append(qualOrs, "(json_extract(details, '$.qualification') LIKE '%Graduate%' OR json_extract(details, '$.qualification') LIKE '%Degree%')")
			} else if strings.Contains(qLower, "ca") || strings.Contains(qLower, "finance") {
				qualOrs = append(qualOrs, "(json_extract(details, '$.qualification') LIKE '%CA%' OR json_extract(details, '$.qualification') LIKE '%Finance%')")
			} else {
				qualOrs = append(qualOrs, "json_extract(details, '$.qualification') LIKE ?")
				args = append(args, "%"+qClean+"%")
			}
		}
		if len(qualOrs) > 0 {
			whereClauses = append(whereClauses, "("+strings.Join(qualOrs, " OR ")+")")
		}
	}

	// 6. States filters
	if len(params.States) > 0 {
		var stateOrs []string
		stateOrs = append(stateOrs, "json_extract(details, '$.state') = 'All India'")
		for _, st := range params.States {
			st = strings.TrimSpace(st)
			if st == "" || strings.EqualFold(st, "all") || strings.EqualFold(st, "all india") {
				continue
			}
			stateOrs = append(stateOrs, "(json_extract(details, '$.state') = ? OR json_extract(details, '$.state') LIKE ?)")
			args = append(args, st, "%"+st+"%")
		}
		whereClauses = append(whereClauses, "("+strings.Join(stateOrs, " OR ")+")")
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Total matching count
	countQuery := d.rebind("SELECT COUNT(1) FROM jobs" + whereSQL)
	var totalCount int
	if err := d.conn.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// 2. Order by
	orderSQL := " ORDER BY COALESCE(published_at, created_at) DESC, id DESC"
	switch strings.ToLower(params.SortBy) {
	case "posts":
		if d.driverName == "postgres" {
			orderSQL = " ORDER BY safe_cast_int(json_extract(details, '$.total_vacancies')) DESC, id DESC"
		} else {
			orderSQL = " ORDER BY CAST(json_extract(details, '$.total_vacancies') AS INTEGER) DESC, id DESC"
		}
	case "deadline":
		orderSQL = " ORDER BY CASE WHEN json_extract(details, '$.application_end') IS NULL OR json_extract(details, '$.application_end') = '' OR json_extract(details, '$.application_end') LIKE '%Refer%' THEN 1 ELSE 0 END, json_extract(details, '$.application_end') ASC, id DESC"
	case "latest":
		orderSQL = " ORDER BY COALESCE(published_at, created_at) DESC, id DESC"
	}

	selectQuery := "SELECT id, hash, source_portal, status, raw_notice, details, review_notes, created_at, updated_at, published_at FROM jobs" + whereSQL + orderSQL

	queryArgs := append([]interface{}{}, args...)
	if params.Limit > 0 {
		selectQuery += " LIMIT ? OFFSET ?"
		queryArgs = append(queryArgs, params.Limit, params.Offset)
	}

	rows, err := d.conn.Query(d.rebind(selectQuery), queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	jobs := make([]models.Job, 0)
	for rows.Next() {
		var j models.Job
		var rawStr, detStr string
		var pubAt sql.NullTime

		err := rows.Scan(&j.ID, &j.Hash, &j.SourcePortal, &j.Status, &rawStr, &detStr, &j.ReviewNotes, &j.CreatedAt, &j.UpdatedAt, &pubAt)
		if err != nil {
			return nil, 0, err
		}

		if pubAt.Valid {
			j.PublishedAt = &pubAt.Time
		}

		_ = json.Unmarshal([]byte(rawStr), &j.RawNotice)
		_ = json.Unmarshal([]byte(detStr), &j.Details)

		jobs = append(jobs, j)
	}

	return jobs, totalCount, nil
}

func (d *DB) GetJobs(status, noticeType, category, state string, limit, offset int) ([]models.Job, int, error) {
	var nTypes, cats, states []string
	if noticeType != "" && noticeType != "all" {
		nTypes = []string{noticeType}
	}
	if category != "" && category != "all" {
		cats = []string{category}
	}
	if state != "" && state != "all" {
		states = []string{state}
	}
	return d.QueryJobs(JobFilterParams{
		Status:      status,
		NoticeTypes: nTypes,
		Categories:  cats,
		States:      states,
		Limit:       limit,
		Offset:      offset,
	})
}

func (d *DB) GetJobsByStatus(status models.JobStatus, noticeType string, limit, offset int) ([]models.Job, error) {
	jobs, _, err := d.GetJobs(string(status), noticeType, "", "", limit, offset)
	return jobs, err
}

func (d *DB) BulkPublish(ids []int64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if len(ids) == 0 {
		query := `UPDATE jobs SET status = ?, published_at = ?, updated_at = ? WHERE status = ?`
		res, err := d.conn.Exec(d.rebind(query), models.StatusPublished, now, now, models.StatusDraft)
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+3)
	args[0] = models.StatusPublished
	args[1] = now
	args[2] = now
	for i, id := range ids {
		placeholders[i] = "?"
		args[i+3] = id
	}

	query := fmt.Sprintf(`UPDATE jobs SET status = ?, published_at = ?, updated_at = ? WHERE id IN (%s)`, strings.Join(placeholders, ","))
	res, err := d.conn.Exec(d.rebind(query), args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (d *DB) SearchJobs(q, qual, state, category, noticeType, status string, limit, offset int) ([]models.Job, int, error) {
	var quals, states, cats, nTypes []string
	if qual != "" {
		quals = []string{qual}
	}
	if state != "" && state != "all" {
		states = []string{state}
	}
	if category != "" && category != "all" {
		cats = []string{category}
	}
	if noticeType != "" && noticeType != "all" {
		nTypes = []string{noticeType}
	}
	return d.QueryJobs(JobFilterParams{
		Status:         status,
		Query:          q,
		NoticeTypes:    nTypes,
		Categories:     cats,
		Qualifications: quals,
		States:         states,
		Limit:          limit,
		Offset:         offset,
	})
}

func (d *DB) GetClosingSoonJobs(days int, limit int) ([]models.Job, error) {
	if limit <= 0 {
		limit = 50
	}
	// Query jobs with valid date string in application_end
	query := `
	SELECT id, hash, source_portal, status, raw_notice, details, review_notes, created_at, updated_at, published_at
	FROM jobs
	WHERE json_extract(details, '$.application_end') != 'Refer notification'
	  AND json_extract(details, '$.application_end') != ''
	  AND json_extract(details, '$.notice_type') = 'recruitment'
	ORDER BY created_at DESC
	LIMIT ?
	`
	rows, err := d.conn.Query(d.rebind(query), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]models.Job, 0)
	for rows.Next() {
		var j models.Job
		var rawStr, detStr string
		var pubAt sql.NullTime

		if err := rows.Scan(&j.ID, &j.Hash, &j.SourcePortal, &j.Status, &rawStr, &detStr, &j.ReviewNotes, &j.CreatedAt, &j.UpdatedAt, &pubAt); err != nil {
			return nil, err
		}
		if pubAt.Valid {
			j.PublishedAt = &pubAt.Time
		}
		_ = json.Unmarshal([]byte(rawStr), &j.RawNotice)
		_ = json.Unmarshal([]byte(detStr), &j.Details)
		jobs = append(jobs, j)
	}

	return jobs, nil
}

func (d *DB) GetAnalytics() (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	var totalJobs int
	_ = d.conn.QueryRow("SELECT COUNT(1) FROM jobs").Scan(&totalJobs)
	analytics["total_jobs"] = totalJobs

	var totalVacancies int
	if d.driverName == "postgres" {
		_ = d.conn.QueryRow("SELECT COALESCE(SUM(safe_cast_int(json_extract(details, '$.total_vacancies'))), 0) FROM jobs").Scan(&totalVacancies)
	} else {
		_ = d.conn.QueryRow("SELECT COALESCE(SUM(CAST(json_extract(details, '$.total_vacancies') AS INTEGER)), 0) FROM jobs").Scan(&totalVacancies)
	}
	analytics["total_vacancies"] = totalVacancies

	// Breakdown by notice type
	typeCounts := make(map[string]int)
	if rows, err := d.conn.Query("SELECT json_extract(details, '$.notice_type'), COUNT(1) FROM jobs GROUP BY json_extract(details, '$.notice_type')"); err == nil {
		defer rows.Close()
		for rows.Next() {
			var nType string
			var count int
			if err := rows.Scan(&nType, &count); err == nil && nType != "" {
				typeCounts[nType] = count
			}
		}
	}
	analytics["notice_types"] = typeCounts

	// Breakdown by category
	categoryCounts := make(map[string]int)
	if rows, err := d.conn.Query("SELECT json_extract(details, '$.category'), COUNT(1) FROM jobs GROUP BY json_extract(details, '$.category') ORDER BY COUNT(1) DESC"); err == nil {
		defer rows.Close()
		for rows.Next() {
			var cat string
			var count int
			if err := rows.Scan(&cat, &count); err == nil && cat != "" {
				categoryCounts[cat] = count
			}
		}
	}
	analytics["categories"] = categoryCounts

	// Top states
	stateCounts := make(map[string]int)
	if rows, err := d.conn.Query("SELECT json_extract(details, '$.state'), COUNT(1) FROM jobs GROUP BY json_extract(details, '$.state') ORDER BY COUNT(1) DESC LIMIT 10"); err == nil {
		defer rows.Close()
		for rows.Next() {
			var st string
			var count int
			if err := rows.Scan(&st, &count); err == nil && st != "" {
				stateCounts[st] = count
			}
		}
	}
	analytics["top_states"] = stateCounts

	// Portal counts
	portalCounts := make(map[string]int)
	if rows, err := d.conn.Query("SELECT source_portal, COUNT(1) FROM jobs GROUP BY source_portal ORDER BY COUNT(1) DESC"); err == nil {
		defer rows.Close()
		for rows.Next() {
			var p string
			var count int
			if err := rows.Scan(&p, &count); err == nil && p != "" {
				portalCounts[p] = count
			}
		}
	}
	analytics["portals"] = portalCounts

	return analytics, nil
}

func (d *DB) GetRecentPublished(limit int) ([]models.Job, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
	SELECT id, hash, source_portal, status, raw_notice, details, review_notes, created_at, updated_at, published_at
	FROM jobs
	WHERE status = 'published' OR status = 'draft'
	ORDER BY created_at DESC
	LIMIT ?
	`
	rows, err := d.conn.Query(d.rebind(query), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]models.Job, 0)
	for rows.Next() {
		var j models.Job
		var rawStr, detStr string
		var pubAt sql.NullTime

		if err := rows.Scan(&j.ID, &j.Hash, &j.SourcePortal, &j.Status, &rawStr, &detStr, &j.ReviewNotes, &j.CreatedAt, &j.UpdatedAt, &pubAt); err != nil {
			return nil, err
		}
		if pubAt.Valid {
			j.PublishedAt = &pubAt.Time
		}
		_ = json.Unmarshal([]byte(rawStr), &j.RawNotice)
		_ = json.Unmarshal([]byte(detStr), &j.Details)
		jobs = append(jobs, j)
	}

	return jobs, nil
}

func (d *DB) GetJobByID(id int64) (*models.Job, error) {
	query := `
	SELECT id, hash, source_portal, status, raw_notice, details, review_notes, created_at, updated_at, published_at
	FROM jobs
	WHERE id = ?
	`
	var j models.Job
	var rawStr, detStr string
	var pubAt sql.NullTime

	err := d.conn.QueryRow(d.rebind(query), id).Scan(&j.ID, &j.Hash, &j.SourcePortal, &j.Status, &rawStr, &detStr, &j.ReviewNotes, &j.CreatedAt, &j.UpdatedAt, &pubAt)
	if err != nil {
		return nil, err
	}

	if pubAt.Valid {
		j.PublishedAt = &pubAt.Time
	}

	_ = json.Unmarshal([]byte(rawStr), &j.RawNotice)
	_ = json.Unmarshal([]byte(detStr), &j.Details)

	return &j, nil
}

func (d *DB) UpdateJob(job *models.Job) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	detailsJSON, err := json.Marshal(job.Details)
	if err != nil {
		return err
	}

	now := time.Now()
	var query string
	if d.driverName == "postgres" {
		query = `
		UPDATE jobs
		SET details = $1::jsonb, review_notes = $2, updated_at = $3
		WHERE id = $4
		`
		_, err = d.conn.Exec(query, string(detailsJSON), job.ReviewNotes, now, job.ID)
	} else {
		query = `
		UPDATE jobs
		SET details = ?, review_notes = ?, updated_at = ?
		WHERE id = ?
		`
		_, err = d.conn.Exec(query, string(detailsJSON), job.ReviewNotes, now, job.ID)
	}
	return err
}

func (d *DB) PublishJob(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	query := `
	UPDATE jobs
	SET status = ?, published_at = ?, updated_at = ?
	WHERE id = ?
	`
	_, err := d.conn.Exec(d.rebind(query), models.StatusPublished, now, now, id)
	return err
}

func (d *DB) RejectJob(id int64, reason string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	query := `
	UPDATE jobs
	SET status = ?, review_notes = ?, updated_at = ?
	WHERE id = ?
	`
	_, err := d.conn.Exec(d.rebind(query), models.StatusRejected, reason, now, id)
	return err
}

func (d *DB) GetJobsForReExtraction() ([]models.Job, error) {
	query := `
	SELECT id, hash, source_portal, status, raw_notice, details, review_notes, created_at, updated_at, published_at
	FROM jobs
	WHERE status = 'draft'
	ORDER BY id ASC
	`
	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]models.Job, 0)
	for rows.Next() {
		var j models.Job
		var rawStr, detStr string
		var pubAt sql.NullTime

		err := rows.Scan(&j.ID, &j.Hash, &j.SourcePortal, &j.Status, &rawStr, &detStr, &j.ReviewNotes, &j.CreatedAt, &j.UpdatedAt, &pubAt)
		if err != nil {
			return nil, err
		}

		if pubAt.Valid {
			j.PublishedAt = &pubAt.Time
		}

		_ = json.Unmarshal([]byte(rawStr), &j.RawNotice)
		_ = json.Unmarshal([]byte(detStr), &j.Details)

		jobs = append(jobs, j)
	}

	return jobs, nil
}

func (d *DB) EnsureDefaultAdmin() error {
	const adminEmail = "admin@admin.com"
	const adminPass = "admin@7077"

	var count int
	err := d.conn.QueryRow(d.rebind("SELECT COUNT(1) FROM users WHERE LOWER(email) = LOWER(?)"), adminEmail).Scan(&count)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()
	if count == 0 {
		if d.driverName == "postgres" {
			_, err = d.conn.Exec(`
				INSERT INTO users (name, email, password_hash, role, phone, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, "System Administrator", adminEmail, string(hash), string(models.RoleAdmin), "", now, now)
		} else {
			_, err = d.conn.Exec(`
				INSERT INTO users (name, email, password_hash, role, phone, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, "System Administrator", adminEmail, string(hash), string(models.RoleAdmin), "", now, now)
		}
		if err != nil {
			return fmt.Errorf("failed to insert default admin: %w", err)
		}
	} else {
		_, err = d.conn.Exec(d.rebind("UPDATE users SET password_hash = ?, role = 'admin', updated_at = ? WHERE LOWER(email) = LOWER(?)"), string(hash), now, adminEmail)
		if err != nil {
			return fmt.Errorf("failed to update default admin: %w", err)
		}
	}
	return nil
}

func (d *DB) CreateUser(u *models.User, rawPassword string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	u.PasswordHash = string(hash)
	if u.Role == "" {
		u.Role = models.RoleUser
	}
	u.CreatedAt = now
	u.UpdatedAt = now

	if d.driverName == "postgres" {
		query := `
		INSERT INTO users (name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
		`
		var id int64
		err := d.conn.QueryRow(query, u.Name, u.Email, u.PasswordHash, string(u.Role), u.Phone, u.GoogleID, u.AvatarURL, u.AuthProvider, now, now).Scan(&id)
		if err != nil {
			return err
		}
		u.ID = id
		return nil
	}

	query := `
	INSERT INTO users (name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := d.conn.Exec(query, u.Name, u.Email, u.PasswordHash, string(u.Role), u.Phone, u.GoogleID, u.AvatarURL, u.AuthProvider, now, now)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		u.ID = id
	}
	return nil
}

func (d *DB) GetUserByEmail(email string) (*models.User, error) {
	query := d.rebind(`
	SELECT id, name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at
	FROM users
	WHERE LOWER(email) = LOWER(?)
	LIMIT 1
	`)
	row := d.conn.QueryRow(query, strings.TrimSpace(email))
	var u models.User
	var roleStr string
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &roleStr, &u.Phone, &u.GoogleID, &u.AvatarURL, &u.AuthProvider, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.Role = models.UserRole(roleStr)
	return &u, nil
}

func (d *DB) GetUserByID(id int64) (*models.User, error) {
	query := d.rebind(`
	SELECT id, name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at
	FROM users
	WHERE id = ?
	LIMIT 1
	`)
	row := d.conn.QueryRow(query, id)
	var u models.User
	var roleStr string
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &roleStr, &u.Phone, &u.GoogleID, &u.AvatarURL, &u.AuthProvider, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.Role = models.UserRole(roleStr)
	return &u, nil
}

// FindOrCreateGoogleUser finds existing user by Google ID or Email, or registers a new user
func (d *DB) FindOrCreateGoogleUser(g *models.GoogleUserInfo) (*models.User, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	email := strings.TrimSpace(g.Email)
	now := time.Now()

	// 1. Check by google_id
	if g.ID != "" {
		query := d.rebind(`
		SELECT id, name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at
		FROM users
		WHERE google_id = ?
		LIMIT 1
		`)
		row := d.conn.QueryRow(query, g.ID)
		var u models.User
		var roleStr string
		err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &roleStr, &u.Phone, &u.GoogleID, &u.AvatarURL, &u.AuthProvider, &u.CreatedAt, &u.UpdatedAt)
		if err == nil {
			// Update picture / name if changed
			if g.Picture != "" && u.AvatarURL != g.Picture {
				u.AvatarURL = g.Picture
				_, _ = d.conn.Exec(d.rebind("UPDATE users SET avatar_url = ?, updated_at = ? WHERE id = ?"), g.Picture, now, u.ID)
			}
			u.Role = models.UserRole(roleStr)
			return &u, nil
		}
	}

	// 2. Check by email
	if email != "" {
		query := d.rebind(`
		SELECT id, name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER(?)
		LIMIT 1
		`)
		row := d.conn.QueryRow(query, email)
		var u models.User
		var roleStr string
		err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &roleStr, &u.Phone, &u.GoogleID, &u.AvatarURL, &u.AuthProvider, &u.CreatedAt, &u.UpdatedAt)
		if err == nil {
			// Link Google ID & Avatar to existing user
			u.GoogleID = g.ID
			if g.Picture != "" {
				u.AvatarURL = g.Picture
			}
			u.AuthProvider = "google"
			u.UpdatedAt = now
			_, _ = d.conn.Exec(d.rebind("UPDATE users SET google_id = ?, avatar_url = ?, auth_provider = 'google', updated_at = ? WHERE id = ?"), g.ID, u.AvatarURL, now, u.ID)
			u.Role = models.UserRole(roleStr)
			return &u, nil
		}
	}

	// 3. Create new Google user
	name := strings.TrimSpace(g.Name)
	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	newUser := &models.User{
		Name:         name,
		Email:        email,
		Role:         models.RoleUser,
		GoogleID:     g.ID,
		AvatarURL:    g.Picture,
		AuthProvider: "google",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if d.driverName == "postgres" {
		query := `
		INSERT INTO users (name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
		`
		var id int64
		err := d.conn.QueryRow(query, newUser.Name, newUser.Email, "", string(newUser.Role), "", newUser.GoogleID, newUser.AvatarURL, newUser.AuthProvider, now, now).Scan(&id)
		if err != nil {
			return nil, err
		}
		newUser.ID = id
		return newUser, nil
	}

	query := `
	INSERT INTO users (name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := d.conn.Exec(query, newUser.Name, newUser.Email, "", string(newUser.Role), "", newUser.GoogleID, newUser.AvatarURL, newUser.AuthProvider, now, now)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err == nil {
		newUser.ID = id
	}

	return newUser, nil
}

func (d *DB) ListUsers(limit, offset int) ([]models.User, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	err := d.conn.QueryRow("SELECT COUNT(1) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := d.rebind(`
	SELECT id, name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at
	FROM users
	ORDER BY id ASC
	LIMIT ? OFFSET ?
	`)
	rows, err := d.conn.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		var roleStr string
		err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &roleStr, &u.Phone, &u.GoogleID, &u.AvatarURL, &u.AuthProvider, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		u.Role = models.UserRole(roleStr)
		users = append(users, u)
	}

	return users, total, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}
