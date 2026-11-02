package job_test

import (
	"path/filepath"
	"testing"
	"time"

	"rojgarbridge-backend/database"
	"rojgarbridge-backend/modules/job"
)

func TestNormalizedJobsSchemaAndQueries(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_jobs_norm.db")
	dbConn, err := database.InitDB(tempDB, "")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer dbConn.Close()

	svc := job.NewService(dbConn, nil)
	if err := svc.EnsurePublishedJobs(); err != nil {
		t.Fatalf("EnsurePublishedJobs failed: %v", err)
	}

	// Insert test fixtures for unit test
	testJobs := []*job.Job{
		{
			Hash:         "test-hash-1",
			SourcePortal: "SSC",
			Status:       job.StatusDraft,
			RawNotice: job.RawNotice{
				SourcePortal: "SSC",
				NoticeTitle:  "SSC Combined Graduate Level Exam",
				FetchedAt:    time.Now(),
			},
			Details: job.ExtractedJobDetails{
				Title:          "SSC Combined Graduate Level Exam",
				NoticeType:     "recruitment",
				Organization:   "Staff Selection Commission",
				Category:       "Central Govt",
				State:          "All India",
				TotalVacancies: 5000,
			},
		},
		{
			Hash:         "test-hash-2",
			SourcePortal: "UPSC",
			Status:       job.StatusDraft,
			RawNotice: job.RawNotice{
				SourcePortal: "UPSC",
				NoticeTitle:  "Civil Services Exam",
				FetchedAt:    time.Now(),
			},
			Details: job.ExtractedJobDetails{
				Title:          "Civil Services Exam",
				NoticeType:     "recruitment",
				Organization:   "UPSC",
				Category:       "Central Govt",
				State:          "All India",
				TotalVacancies: 1000,
			},
		},
	}

	for _, j := range testJobs {
		if err := svc.SaveDraft(j); err != nil {
			t.Fatalf("SaveDraft failed: %v", err)
		}
		if _, err := svc.PublishJob(j.ID); err != nil {
			t.Fatalf("PublishJob failed: %v", err)
		}
	}

	// 1. Verify published jobs
	jobs, total, err := svc.QueryJobs(job.JobFilterParams{
		Status: "published",
	})
	if err != nil {
		t.Fatalf("QueryJobs failed: %v", err)
	}
	if total == 0 || len(jobs) == 0 {
		t.Fatalf("Expected published jobs, got 0")
	}

	// 2. Test normalized column search (e.g. query "SSC")
	sscJobs, sscTotal, err := svc.QueryJobs(job.JobFilterParams{
		Status: "published",
		Query:  "SSC",
	})
	if err != nil {
		t.Fatalf("QueryJobs for SSC failed: %v", err)
	}
	if sscTotal == 0 || len(sscJobs) == 0 {
		t.Fatalf("Expected at least 1 SSC job, got 0")
	}

	// 3. Test sorting by posts (total_vacancies DESC)
	sortedJobs, _, err := svc.QueryJobs(job.JobFilterParams{
		Status: "published",
		SortBy: "posts",
	})
	if err != nil {
		t.Fatalf("QueryJobs sort by posts failed: %v", err)
	}
	if len(sortedJobs) > 1 {
		if sortedJobs[0].Details.TotalVacancies < sortedJobs[1].Details.TotalVacancies {
			t.Fatalf("Expected sortedJobs[0].Vacancies >= sortedJobs[1].Vacancies, got %d < %d",
				sortedJobs[0].Details.TotalVacancies, sortedJobs[1].Details.TotalVacancies)
		}
	}

	// 4. Test GetAnalytics
	analytics, err := svc.GetAnalytics()
	if err != nil {
		t.Fatalf("GetAnalytics failed: %v", err)
	}
	if analytics["total_jobs"].(int) == 0 {
		t.Fatalf("Expected total_jobs > 0, got 0")
	}
	if analytics["total_vacancies"].(int) == 0 {
		t.Fatalf("Expected total_vacancies > 0, got 0")
	}
}
