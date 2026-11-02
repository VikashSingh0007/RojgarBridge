package job

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"rojgarbridge-backend/database"
	"rojgarbridge-backend/utils"
)

type Service interface {
	QueryJobs(params JobFilterParams) ([]Job, int, error)
	GetJobByID(id int64) (*Job, error)
	UpdateJob(job *Job) error
	PublishJob(id int64) (*Job, error)
	BulkPublish(ids []int64) (int64, error)
	RejectJob(id int64, reason string) error
	GetClosingSoonJobs(days int, limit int) ([]Job, error)
	GetAnalytics() (map[string]interface{}, error)
	GetRecentPublished(limit int) ([]Job, error)
	SaveDraft(job *Job) error
	ExistsByHash(hash string) (bool, error)
	GetJobsForReExtraction() ([]Job, error)
	EnsurePublishedJobs() error
	SeedInitialVerifiedJobs() error
}

type service struct {
	db       *database.DB
	notifier utils.Notifier
}

func NewService(db *database.DB, notif utils.Notifier) Service {
	return &service{
		db:       db,
		notifier: notif,
	}
}

func (s *service) ExistsByHash(hash string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(1) FROM jobs WHERE hash = ?", hash).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *service) SaveDraft(job *Job) error {
	s.db.Lock()
	defer s.db.Unlock()

	rawJSON, err := json.Marshal(job.RawNotice)
	if err != nil {
		return fmt.Errorf("failed to marshal raw notice: %w", err)
	}

	detailsJSON, err := json.Marshal(job.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	det := job.Details
	nType := det.NoticeType
	if nType == "" {
		nType = "recruitment"
	}
	state := det.State
	if state == "" {
		state = "All India"
	}
	confScore := det.ConfidenceScore
	if confScore <= 0 {
		confScore = 1.0
	}

	now := time.Now()
	if s.db.DriverName() == "postgres" {
		query := `
		INSERT INTO jobs (
			hash, source_portal, status,
			title, notice_type, organization, category, state, advt_number,
			total_vacancies, min_age, max_age, qualification, application_start,
			application_end, exam_date, fee_general, fee_reserved, salary_payscale,
			official_apply_url, official_notice_url, confidence_score,
			raw_notice, details, review_notes, created_at, updated_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19,
			$20, $21, $22,
			$23::jsonb, $24::jsonb, $25, $26, $27
		) RETURNING id
		`
		var id int64
		err := s.db.QueryRow(query,
			job.Hash, job.SourcePortal, StatusDraft,
			det.Title, nType, det.Organization, det.Category, state, det.AdvtNumber,
			det.TotalVacancies, det.MinAge, det.MaxAge, det.Qualification, det.ApplicationStart,
			det.ApplicationEnd, det.ExamDate, det.FeeGeneral, det.FeeReserved, det.SalaryPayScale,
			det.OfficialApplyURL, det.OfficialNoticeURL, confScore,
			string(rawJSON), string(detailsJSON), job.ReviewNotes, now, now,
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("failed to insert draft: %w", err)
		}
		job.ID = id
		job.Status = StatusDraft
		job.CreatedAt = now
		job.UpdatedAt = now
		return nil
	}

	query := `
	INSERT INTO jobs (
		hash, source_portal, status,
		title, notice_type, organization, category, state, advt_number,
		total_vacancies, min_age, max_age, qualification, application_start,
		application_end, exam_date, fee_general, fee_reserved, salary_payscale,
		official_apply_url, official_notice_url, confidence_score,
		raw_notice, details, review_notes, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := s.db.Exec(query,
		job.Hash, job.SourcePortal, StatusDraft,
		det.Title, nType, det.Organization, det.Category, state, det.AdvtNumber,
		det.TotalVacancies, det.MinAge, det.MaxAge, det.Qualification, det.ApplicationStart,
		det.ApplicationEnd, det.ExamDate, det.FeeGeneral, det.FeeReserved, det.SalaryPayScale,
		det.OfficialApplyURL, det.OfficialNoticeURL, confScore,
		string(rawJSON), string(detailsJSON), job.ReviewNotes, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert draft: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		job.ID = id
		job.Status = StatusDraft
		job.CreatedAt = now
		job.UpdatedAt = now
	}
	return nil
}

func (s *service) QueryJobs(params JobFilterParams) ([]Job, int, error) {
	var whereClauses []string
	var args []interface{}

	// 1. Status (published, draft, etc.)
	if params.Status != "" && params.Status != "all" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, params.Status)
	}

	// 2. Search query across indexed columns
	if strings.TrimSpace(params.Query) != "" {
		term := "%" + strings.TrimSpace(params.Query) + "%"
		if s.db.DriverName() == "postgres" {
			whereClauses = append(whereClauses, "(title ILIKE ? OR organization ILIKE ? OR advt_number ILIKE ? OR qualification ILIKE ? OR category ILIKE ? OR state ILIKE ? OR source_portal ILIKE ?)")
		} else {
			whereClauses = append(whereClauses, "(title LIKE ? OR organization LIKE ? OR advt_number LIKE ? OR qualification LIKE ? OR category LIKE ? OR state LIKE ? OR source_portal LIKE ?)")
		}
		args = append(args, term, term, term, term, term, term, term)
	}

	// 3. Notice types filter
	if len(params.NoticeTypes) > 0 {
		jsonTypeExpr := "details->>'notice_type'"
		if s.db.DriverName() != "postgres" {
			jsonTypeExpr = "json_extract(details, '$.notice_type')"
		}

		var typeOrs []string
		for _, t := range params.NoticeTypes {
			tClean := strings.TrimSpace(strings.ToLower(t))
			if tClean == "" || tClean == "all" {
				continue
			}
			if strings.Contains(tClean, "recruitment") || tClean == "job" || tClean == "jobs" {
				typeOrs = append(typeOrs, fmt.Sprintf("(notice_type = 'recruitment' OR notice_type IS NULL OR notice_type = '' OR %s = 'recruitment')", jsonTypeExpr))
			} else if strings.Contains(tClean, "announcement") {
				typeOrs = append(typeOrs, fmt.Sprintf("(notice_type IN ('admit_card', 'result', 'answer_key', 'interview_schedule') OR %s IN ('admit_card', 'result', 'answer_key', 'interview_schedule'))", jsonTypeExpr))
			} else if strings.Contains(tClean, "admit") {
				typeOrs = append(typeOrs, fmt.Sprintf("(notice_type = 'admit_card' OR %s = 'admit_card')", jsonTypeExpr))
			} else if strings.Contains(tClean, "result") {
				typeOrs = append(typeOrs, fmt.Sprintf("(notice_type = 'result' OR %s = 'result')", jsonTypeExpr))
			} else if strings.Contains(tClean, "answer") {
				typeOrs = append(typeOrs, fmt.Sprintf("(notice_type = 'answer_key' OR %s = 'answer_key')", jsonTypeExpr))
			} else if strings.Contains(tClean, "interview") || strings.Contains(tClean, "schedule") || strings.Contains(tClean, "exam") {
				typeOrs = append(typeOrs, fmt.Sprintf("(notice_type = 'interview_schedule' OR %s = 'interview_schedule')", jsonTypeExpr))
			} else {
				typeOrs = append(typeOrs, fmt.Sprintf("(notice_type = ? OR %s = ?)", jsonTypeExpr))
				args = append(args, tClean, tClean)
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
				catOrs = append(catOrs, "(category LIKE '%Central%' OR source_portal IN ('UPSC', 'SSC', 'RRB', 'IBPS', 'ISRO'))")
			} else if cLower == "state govt" {
				catOrs = append(catOrs, "(category LIKE '%State%' OR source_portal IN ('UPPSC', 'BPSC', 'MPPSC', 'RPSC'))")
			} else if cLower == "banking" || cLower == "bank" {
				catOrs = append(catOrs, "(category LIKE '%Bank%' OR organization LIKE '%Bank%' OR source_portal = 'IBPS')")
			} else if strings.Contains(cLower, "defence") || strings.Contains(cLower, "police") {
				catOrs = append(catOrs, "(category LIKE '%Defence%' OR category LIKE '%Police%' OR title LIKE '%Police%' OR title LIKE '%Defence%')")
			} else if strings.Contains(cLower, "teach") {
				catOrs = append(catOrs, "(category LIKE '%Teach%' OR title LIKE '%Teacher%' OR title LIKE '%Lecturer%' OR title LIKE '%Principal%')")
			} else if strings.Contains(cLower, "engineering") || strings.Contains(cLower, "psu") {
				catOrs = append(catOrs, "(category LIKE '%Engineering%' OR category LIKE '%PSU%' OR title LIKE '%Engineer%' OR source_portal = 'ISRO')")
			} else if cLower == "railway" {
				catOrs = append(catOrs, "(source_portal = 'RRB' OR organization LIKE '%Railway%')")
			} else if cLower == "ssc" {
				catOrs = append(catOrs, "(source_portal = 'SSC' OR organization LIKE '%Staff Selection%')")
			} else if cLower == "upsc" {
				catOrs = append(catOrs, "(source_portal = 'UPSC' OR organization LIKE '%UPSC%')")
			} else {
				catOrs = append(catOrs, "(category LIKE ? OR organization LIKE ? OR source_portal = ?)")
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
				qualOrs = append(qualOrs, "(qualification LIKE '%10th%' OR qualification LIKE '%Matric%')")
			} else if strings.Contains(qLower, "12th") {
				qualOrs = append(qualOrs, "(qualification LIKE '%12th%' OR qualification LIKE '%Intermediate%' OR qualification LIKE '%10+2%')")
			} else if strings.Contains(qLower, "iti") {
				qualOrs = append(qualOrs, "qualification LIKE '%ITI%'")
			} else if strings.Contains(qLower, "diploma") {
				qualOrs = append(qualOrs, "qualification LIKE '%Diploma%'")
			} else if strings.Contains(qLower, "b.tech") || strings.Contains(qLower, "b.e") {
				qualOrs = append(qualOrs, "(qualification LIKE '%B.E%' OR qualification LIKE '%B.Tech%' OR qualification LIKE '%Engineer%')")
			} else if strings.Contains(qLower, "post graduate") || strings.Contains(qLower, "pg") {
				qualOrs = append(qualOrs, "(qualification LIKE '%Post Graduate%' OR qualification LIKE '%PG%' OR qualification LIKE '%Master%')")
			} else if strings.Contains(qLower, "graduate") {
				qualOrs = append(qualOrs, "(qualification LIKE '%Graduate%' OR qualification LIKE '%Degree%')")
			} else if strings.Contains(qLower, "ca") || strings.Contains(qLower, "finance") {
				qualOrs = append(qualOrs, "(qualification LIKE '%CA%' OR qualification LIKE '%Finance%')")
			} else {
				qualOrs = append(qualOrs, "qualification LIKE ?")
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
		stateOrs = append(stateOrs, "state = 'All India'")
		for _, st := range params.States {
			st = strings.TrimSpace(st)
			if st == "" || strings.EqualFold(st, "all") || strings.EqualFold(st, "all india") {
				continue
			}
			stateOrs = append(stateOrs, "(state = ? OR state LIKE ?)")
			args = append(args, st, "%"+st+"%")
		}
		whereClauses = append(whereClauses, "("+strings.Join(stateOrs, " OR ")+")")
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Total matching count (uses index)
	countQuery := "SELECT COUNT(1) FROM jobs" + whereSQL
	var totalCount int
	if err := s.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// 2. Order by
	orderSQL := " ORDER BY COALESCE(published_at, created_at) DESC, id DESC"
	switch strings.ToLower(params.SortBy) {
	case "posts":
		orderSQL = " ORDER BY total_vacancies DESC, id DESC"
	case "deadline":
		orderSQL = " ORDER BY CASE WHEN application_end IS NULL OR application_end = '' OR application_end LIKE '%Refer%' THEN 1 ELSE 0 END, application_end ASC, id DESC"
	case "latest":
		orderSQL = " ORDER BY COALESCE(published_at, created_at) DESC, id DESC"
	}

	selectQuery := "SELECT id, hash, source_portal, status, raw_notice, details, review_notes, created_at, updated_at, published_at FROM jobs" + whereSQL + orderSQL

	queryArgs := append([]interface{}{}, args...)
	if params.Limit > 0 {
		selectQuery += " LIMIT ? OFFSET ?"
		queryArgs = append(queryArgs, params.Limit, params.Offset)
	}

	rows, err := s.db.Query(selectQuery, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	jobs := make([]Job, 0)
	for rows.Next() {
		var j Job
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

func (s *service) BulkPublish(ids []int64) (int64, error) {
	s.db.Lock()
	defer s.db.Unlock()

	now := time.Now()
	if len(ids) == 0 {
		query := `UPDATE jobs SET status = ?, published_at = ?, updated_at = ? WHERE status = ?`
		res, err := s.db.Exec(query, StatusPublished, now, now, StatusDraft)
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+3)
	args[0] = StatusPublished
	args[1] = now
	args[2] = now
	for i, id := range ids {
		placeholders[i] = "?"
		args[i+3] = id
	}

	query := fmt.Sprintf(`UPDATE jobs SET status = ?, published_at = ?, updated_at = ? WHERE id IN (%s)`, strings.Join(placeholders, ","))
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *service) GetClosingSoonJobs(days int, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
	SELECT id, hash, source_portal, status, raw_notice, details, review_notes, created_at, updated_at, published_at
	FROM jobs
	WHERE application_end != 'Refer notification'
	  AND application_end != ''
	  AND notice_type = 'recruitment'
	ORDER BY application_end ASC, created_at DESC
	LIMIT ?
	`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]Job, 0)
	for rows.Next() {
		var j Job
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

func (s *service) GetAnalytics() (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	var totalJobs int
	_ = s.db.QueryRow("SELECT COUNT(1) FROM jobs").Scan(&totalJobs)
	analytics["total_jobs"] = totalJobs

	var totalVacancies int
	_ = s.db.QueryRow("SELECT COALESCE(SUM(total_vacancies), 0) FROM jobs").Scan(&totalVacancies)
	analytics["total_vacancies"] = totalVacancies

	typeCounts := make(map[string]int)
	if rows, err := s.db.Query("SELECT notice_type, COUNT(1) FROM jobs GROUP BY notice_type"); err == nil {
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

	categoryCounts := make(map[string]int)
	if rows, err := s.db.Query("SELECT category, COUNT(1) FROM jobs GROUP BY category ORDER BY COUNT(1) DESC"); err == nil {
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

	stateCounts := make(map[string]int)
	if rows, err := s.db.Query("SELECT state, COUNT(1) FROM jobs GROUP BY state ORDER BY COUNT(1) DESC LIMIT 10"); err == nil {
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

	portalCounts := make(map[string]int)
	if rows, err := s.db.Query("SELECT source_portal, COUNT(1) FROM jobs GROUP BY source_portal ORDER BY COUNT(1) DESC"); err == nil {
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

func (s *service) GetRecentPublished(limit int) ([]Job, error) {
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
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]Job, 0)
	for rows.Next() {
		var j Job
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

func (s *service) GetJobByID(id int64) (*Job, error) {
	query := `
	SELECT id, hash, source_portal, status, raw_notice, details, review_notes, created_at, updated_at, published_at
	FROM jobs
	WHERE id = ?
	`
	var j Job
	var rawStr, detStr string
	var pubAt sql.NullTime

	err := s.db.QueryRow(query, id).Scan(&j.ID, &j.Hash, &j.SourcePortal, &j.Status, &rawStr, &detStr, &j.ReviewNotes, &j.CreatedAt, &j.UpdatedAt, &pubAt)
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

func (s *service) UpdateJob(job *Job) error {
	s.db.Lock()
	defer s.db.Unlock()

	detailsJSON, err := json.Marshal(job.Details)
	if err != nil {
		return err
	}

	det := job.Details
	nType := det.NoticeType
	if nType == "" {
		nType = "recruitment"
	}
	state := det.State
	if state == "" {
		state = "All India"
	}
	confScore := det.ConfidenceScore
	if confScore <= 0 {
		confScore = 1.0
	}

	now := time.Now()
	if s.db.DriverName() == "postgres" {
		query := `
		UPDATE jobs
		SET title = $1, notice_type = $2, organization = $3, category = $4, state = $5, advt_number = $6,
		    total_vacancies = $7, min_age = $8, max_age = $9, qualification = $10, application_start = $11,
		    application_end = $12, exam_date = $13, fee_general = $14, fee_reserved = $15, salary_payscale = $16,
		    official_apply_url = $17, official_notice_url = $18, confidence_score = $19,
		    details = $20::jsonb, review_notes = $21, updated_at = $22
		WHERE id = $23
		`
		_, err = s.db.Exec(query,
			det.Title, nType, det.Organization, det.Category, state, det.AdvtNumber,
			det.TotalVacancies, det.MinAge, det.MaxAge, det.Qualification, det.ApplicationStart,
			det.ApplicationEnd, det.ExamDate, det.FeeGeneral, det.FeeReserved, det.SalaryPayScale,
			det.OfficialApplyURL, det.OfficialNoticeURL, confScore,
			string(detailsJSON), job.ReviewNotes, now, job.ID,
		)
		return err
	}

	query := `
	UPDATE jobs
	SET title = ?, notice_type = ?, organization = ?, category = ?, state = ?, advt_number = ?,
	    total_vacancies = ?, min_age = ?, max_age = ?, qualification = ?, application_start = ?,
	    application_end = ?, exam_date = ?, fee_general = ?, fee_reserved = ?, salary_payscale = ?,
	    official_apply_url = ?, official_notice_url = ?, confidence_score = ?,
	    details = ?, review_notes = ?, updated_at = ?
	WHERE id = ?
	`
	_, err = s.db.Exec(query,
		det.Title, nType, det.Organization, det.Category, state, det.AdvtNumber,
		det.TotalVacancies, det.MinAge, det.MaxAge, det.Qualification, det.ApplicationStart,
		det.ApplicationEnd, det.ExamDate, det.FeeGeneral, det.FeeReserved, det.SalaryPayScale,
		det.OfficialApplyURL, det.OfficialNoticeURL, confScore,
		string(detailsJSON), job.ReviewNotes, now, job.ID,
	)
	return err
}

func (s *service) PublishJob(id int64) (*Job, error) {
	s.db.Lock()
	defer s.db.Unlock()

	job, err := s.GetJobByID(id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	query := `
	UPDATE jobs
	SET status = ?, published_at = ?, updated_at = ?
	WHERE id = ?
	`
	_, err = s.db.Exec(query, StatusPublished, now, now, id)
	if err != nil {
		return nil, err
	}

	job.Status = StatusPublished
	job.PublishedAt = &now
	job.UpdatedAt = now

	// Broadcast notification to Telegram
	if s.notifier != nil && s.notifier.IsEnabled() {
		go func(j *Job) {
			_ = s.notifier.NotifyJobPublished(utils.NotificationJob{
				Title:            j.Details.Title,
				Organization:     j.Details.Organization,
				Category:         j.Details.Category,
				State:            j.Details.State,
				TotalVacancies:   j.Details.TotalVacancies,
				Qualification:    j.Details.Qualification,
				ApplicationEnd:   j.Details.ApplicationEnd,
				OfficialApplyURL: j.Details.OfficialApplyURL,
			})
		}(job)
	}

	return job, nil
}

func (s *service) RejectJob(id int64, reason string) error {
	s.db.Lock()
	defer s.db.Unlock()

	now := time.Now()
	query := `
	UPDATE jobs
	SET status = ?, review_notes = ?, updated_at = ?
	WHERE id = ?
	`
	_, err := s.db.Exec(query, StatusRejected, reason, now, id)
	return err
}

func (s *service) GetJobsForReExtraction() ([]Job, error) {
	query := `
	SELECT id, hash, source_portal, status, raw_notice, details, review_notes, created_at, updated_at, published_at
	FROM jobs
	WHERE status = 'draft'
	ORDER BY id ASC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]Job, 0)
	for rows.Next() {
		var j Job
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

func (s *service) EnsurePublishedJobs() error {
	var pubCount int
	_ = s.db.QueryRow("SELECT COUNT(1) FROM jobs WHERE status = 'published'").Scan(&pubCount)
	log.Printf("[Job Service] Initial check: %d published notifications found in database.", pubCount)
	return nil
}

func (s *service) SeedInitialVerifiedJobs() error {
	// Obsolete: All data must be discovered, fetched, and parsed dynamically from live sources.
	// Hardcoded or seed data is strictly disallowed.
	return nil
}
