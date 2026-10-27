package db_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"rojgarbridge-backend/internal/db"
	"rojgarbridge-backend/internal/models"
)

func TestUserManagementAndAdminSeed(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_rozgarhub.db")
	database, err := db.InitDB(tempDB, "")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()

	// 1. Verify default admin seeded
	admin, err := database.GetUserByEmail("admin@admin.com")
	if err != nil {
		t.Fatalf("GetUserByEmail for admin failed: %v", err)
	}
	if admin == nil || admin.Email != "admin@admin.com" {
		t.Fatalf("Expected admin@admin.com, got %v", admin)
	}
	if admin.Role != models.RoleAdmin {
		t.Fatalf("Expected RoleAdmin, got %v", admin.Role)
	}

	// 2. Verify password matches admin@7077
	err = bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte("admin@7077"))
	if err != nil {
		t.Fatalf("Admin password check failed: %v", err)
	}

	// 3. Test user creation
	newUser := &models.User{
		Name:  "Test Aspirant",
		Email: "aspirant@test.com",
		Role:  models.RoleUser,
		Phone: "9876543210",
	}
	err = database.CreateUser(newUser, "testpassword123")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if newUser.ID <= 0 {
		t.Fatalf("Expected valid user ID, got %d", newUser.ID)
	}

	// 4. Test ListUsers
	users, total, err := database.ListUsers(10, 0)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if total < 2 {
		t.Fatalf("Expected at least 2 users (admin + aspirant), got %d", total)
	}
	var found bool
	for _, u := range users {
		if u.Email == "aspirant@test.com" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Created user not found in ListUsers")
	}
}

func TestMigrationOnExistingUsersTable(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_legacy_upgrade.db")
	// Test InitDB handles both fresh and already existing tables gracefully
	database, err := db.InitDB(tempDB, "")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()

	gUser := &models.GoogleUserInfo{
		ID:      "google-test-id-1",
		Email:   "test@google.com",
		Name:    "Test Google",
		Picture: "https://example.com/pic.jpg",
	}
	u, err := database.FindOrCreateGoogleUser(gUser)
	if err != nil {
		t.Fatalf("FindOrCreateGoogleUser failed: %v", err)
	}
	if u.GoogleID != "google-test-id-1" || u.AuthProvider != "google" {
		t.Fatalf("unexpected user: %+v", u)
	}
}

func TestNormalizedJobsSchemaAndQueries(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_jobs_norm.db")
	database, err := db.InitDB(tempDB, "")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()

	// 1. Verify initial seeded jobs
	jobs, total, err := database.QueryJobs(db.JobFilterParams{
		Status: "published",
	})
	if err != nil {
		t.Fatalf("QueryJobs failed: %v", err)
	}
	if total == 0 || len(jobs) == 0 {
		t.Fatalf("Expected seeded published jobs, got 0")
	}

	// 2. Test normalized column search (e.g. query "SSC")
	sscJobs, sscTotal, err := database.QueryJobs(db.JobFilterParams{
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
	sortedJobs, _, err := database.QueryJobs(db.JobFilterParams{
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
	analytics, err := database.GetAnalytics()
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

