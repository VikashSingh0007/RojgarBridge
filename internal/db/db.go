package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"rojgarbridge-backend/internal/models"
)

type DB struct {
	conn *sql.DB
	mu   sync.Mutex
}

func InitDB(dbPath string) (*DB, error) {
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

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}

func (d *DB) migrate() error {
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
	`
	_, err := d.conn.Exec(query)
	return err
}

func (d *DB) ExistsByHash(hash string) (bool, error) {
	var count int
	err := d.conn.QueryRow("SELECT COUNT(1) FROM jobs WHERE hash = ?", hash).Scan(&count)
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

func (d *DB) GetJobs(status, noticeType, category, state string, limit, offset int) ([]models.Job, int, error) {
	var whereClauses []string
	var args []interface{}

	if status != "" && status != "all" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, status)
	}
	if noticeType != "" && noticeType != "all" {
		whereClauses = append(whereClauses, "json_extract(details, '$.notice_type') = ?")
		args = append(args, noticeType)
	}
	if category != "" && category != "all" {
		whereClauses = append(whereClauses, "json_extract(details, '$.category') = ?")
		args = append(args, category)
	}
	if state != "" && state != "all" {
		whereClauses = append(whereClauses, "json_extract(details, '$.state') = ?")
		args = append(args, state)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Get total count
	countQuery := "SELECT COUNT(1) FROM jobs" + whereSQL
	var totalCount int
	if err := d.conn.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// 2. Build select query
	selectQuery := "SELECT id, hash, source_portal, status, raw_notice, details, review_notes, created_at, updated_at, published_at FROM jobs" + whereSQL + " ORDER BY created_at DESC"

	queryArgs := append([]interface{}{}, args...)
	if limit > 0 {
		selectQuery += " LIMIT ? OFFSET ?"
		queryArgs = append(queryArgs, limit, offset)
	}

	rows, err := d.conn.Query(selectQuery, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []models.Job
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

func (d *DB) GetJobsByStatus(status models.JobStatus, noticeType string, limit, offset int) ([]models.Job, error) {
	jobs, _, err := d.GetJobs(string(status), noticeType, "", "", limit, offset)
	return jobs, err
}

func (d *DB) BulkPublish(ids []int64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if len(ids) == 0 {
		// Publish all drafts
		query := `UPDATE jobs SET status = ?, published_at = ?, updated_at = ? WHERE status = ?`
		res, err := d.conn.Exec(query, models.StatusPublished, now, now, models.StatusDraft)
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}

	// Publish specific IDs
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
	res, err := d.conn.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
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

	err := d.conn.QueryRow(query, id).Scan(&j.ID, &j.Hash, &j.SourcePortal, &j.Status, &rawStr, &detStr, &j.ReviewNotes, &j.CreatedAt, &j.UpdatedAt, &pubAt)
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
	query := `
	UPDATE jobs
	SET details = ?, review_notes = ?, updated_at = ?
	WHERE id = ?
	`
	_, err = d.conn.Exec(query, string(detailsJSON), job.ReviewNotes, now, job.ID)
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
	_, err := d.conn.Exec(query, models.StatusPublished, now, now, id)
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
	_, err := d.conn.Exec(query, models.StatusRejected, reason, now, id)
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

	var jobs []models.Job
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

func (d *DB) Close() error {
	return d.conn.Close()
}
