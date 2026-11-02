package user_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"rojgarbridge-backend/database"
	"rojgarbridge-backend/modules/user"
)

func TestUserManagementAndAdminSeed(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_user_mod.db")
	dbConn, err := database.InitDB(tempDB, "")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer dbConn.Close()

	svc := user.NewService(dbConn)
	if err := svc.EnsureDefaultAdmin(); err != nil {
		t.Fatalf("EnsureDefaultAdmin failed: %v", err)
	}

	// 1. Verify default admin seeded
	admin, err := svc.GetUserByEmail("admin@admin.com")
	if err != nil {
		t.Fatalf("GetUserByEmail for admin failed: %v", err)
	}
	if admin == nil || admin.Email != "admin@admin.com" {
		t.Fatalf("Expected admin@admin.com, got %v", admin)
	}
	if admin.Role != user.RoleAdmin {
		t.Fatalf("Expected RoleAdmin, got %v", admin.Role)
	}

	// 2. Verify password matches admin@7077
	err = bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte("admin@7077"))
	if err != nil {
		t.Fatalf("Admin password check failed: %v", err)
	}

	// 3. Test user creation
	newUser := &user.User{
		Name:  "Test Aspirant",
		Email: "aspirant@test.com",
		Role:  user.RoleUser,
		Phone: "9876543210",
	}
	err = svc.CreateUser(newUser, "testpassword123")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if newUser.ID <= 0 {
		t.Fatalf("Expected valid user ID, got %d", newUser.ID)
	}

	// 4. Test ListUsers
	users, total, err := svc.ListUsers(10, 0)
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

	// 5. Test Google user find or create
	gUser := &user.GoogleUserInfo{
		ID:      "google-test-id-1",
		Email:   "test@google.com",
		Name:    "Test Google",
		Picture: "https://example.com/pic.jpg",
	}
	u, err := svc.FindOrCreateGoogleUser(gUser)
	if err != nil {
		t.Fatalf("FindOrCreateGoogleUser failed: %v", err)
	}
	if u.GoogleID != "google-test-id-1" || u.AuthProvider != "google" {
		t.Fatalf("unexpected user: %+v", u)
	}
}
