package user

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"rojgarbridge-backend/database"
)

type Service interface {
	CreateUser(u *User, rawPassword string) error
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id int64) (*User, error)
	ListUsers(limit, offset int) ([]User, int, error)
	FindOrCreateGoogleUser(g *GoogleUserInfo) (*User, error)
	EnsureDefaultAdmin() error
}

type service struct {
	db *database.DB
}

func NewService(db *database.DB) Service {
	return &service{db: db}
}

func (s *service) EnsureDefaultAdmin() error {
	const adminEmail = "admin@admin.com"
	const adminPass = "admin@7077"

	var count int
	err := s.db.QueryRow("SELECT COUNT(1) FROM users WHERE LOWER(email) = LOWER(?)", adminEmail).Scan(&count)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()
	if count == 0 {
		if s.db.DriverName() == "postgres" {
			_, err = s.db.Exec(`
				INSERT INTO users (name, email, password_hash, role, phone, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, "System Administrator", adminEmail, string(hash), string(RoleAdmin), "", now, now)
		} else {
			_, err = s.db.Exec(`
				INSERT INTO users (name, email, password_hash, role, phone, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, "System Administrator", adminEmail, string(hash), string(RoleAdmin), "", now, now)
		}
		if err != nil {
			return fmt.Errorf("failed to insert default admin: %w", err)
		}
	} else {
		_, err = s.db.Exec("UPDATE users SET password_hash = ?, role = 'admin', updated_at = ? WHERE LOWER(email) = LOWER(?)", string(hash), now, adminEmail)
		if err != nil {
			return fmt.Errorf("failed to update default admin: %w", err)
		}
	}
	return nil
}

func (s *service) CreateUser(u *User, rawPassword string) error {
	s.db.Lock()
	defer s.db.Unlock()

	hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	u.PasswordHash = string(hash)
	if u.Role == "" {
		u.Role = RoleUser
	}
	u.CreatedAt = now
	u.UpdatedAt = now

	if s.db.DriverName() == "postgres" {
		query := `
		INSERT INTO users (name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
		`
		var id int64
		err := s.db.QueryRow(query, u.Name, u.Email, u.PasswordHash, string(u.Role), u.Phone, u.GoogleID, u.AvatarURL, u.AuthProvider, now, now).Scan(&id)
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
	res, err := s.db.Exec(query, u.Name, u.Email, u.PasswordHash, string(u.Role), u.Phone, u.GoogleID, u.AvatarURL, u.AuthProvider, now, now)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		u.ID = id
	}
	return nil
}

func (s *service) GetUserByEmail(email string) (*User, error) {
	query := `
	SELECT id, name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at
	FROM users
	WHERE LOWER(email) = LOWER(?)
	LIMIT 1
	`
	row := s.db.QueryRow(query, strings.TrimSpace(email))
	var u User
	var roleStr string
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &roleStr, &u.Phone, &u.GoogleID, &u.AvatarURL, &u.AuthProvider, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.Role = UserRole(roleStr)
	return &u, nil
}

func (s *service) GetUserByID(id int64) (*User, error) {
	query := `
	SELECT id, name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at
	FROM users
	WHERE id = ?
	LIMIT 1
	`
	row := s.db.QueryRow(query, id)
	var u User
	var roleStr string
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &roleStr, &u.Phone, &u.GoogleID, &u.AvatarURL, &u.AuthProvider, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.Role = UserRole(roleStr)
	return &u, nil
}

func (s *service) FindOrCreateGoogleUser(g *GoogleUserInfo) (*User, error) {
	s.db.Lock()
	defer s.db.Unlock()

	email := strings.TrimSpace(g.Email)
	now := time.Now()

	// 1. Check by google_id
	if g.ID != "" {
		query := `
		SELECT id, name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at
		FROM users
		WHERE google_id = ?
		LIMIT 1
		`
		row := s.db.QueryRow(query, g.ID)
		var u User
		var roleStr string
		err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &roleStr, &u.Phone, &u.GoogleID, &u.AvatarURL, &u.AuthProvider, &u.CreatedAt, &u.UpdatedAt)
		if err == nil {
			if g.Picture != "" && u.AvatarURL != g.Picture {
				u.AvatarURL = g.Picture
				_, _ = s.db.Exec("UPDATE users SET avatar_url = ?, updated_at = ? WHERE id = ?", g.Picture, now, u.ID)
			}
			u.Role = UserRole(roleStr)
			return &u, nil
		}
	}

	// 2. Check by email
	if email != "" {
		query := `
		SELECT id, name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER(?)
		LIMIT 1
		`
		row := s.db.QueryRow(query, email)
		var u User
		var roleStr string
		err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &roleStr, &u.Phone, &u.GoogleID, &u.AvatarURL, &u.AuthProvider, &u.CreatedAt, &u.UpdatedAt)
		if err == nil {
			u.GoogleID = g.ID
			if g.Picture != "" {
				u.AvatarURL = g.Picture
			}
			u.AuthProvider = "google"
			u.UpdatedAt = now
			_, _ = s.db.Exec("UPDATE users SET google_id = ?, avatar_url = ?, auth_provider = 'google', updated_at = ? WHERE id = ?", g.ID, u.AvatarURL, now, u.ID)
			u.Role = UserRole(roleStr)
			return &u, nil
		}
	}

	// 3. Create new Google user
	name := strings.TrimSpace(g.Name)
	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	newUser := &User{
		Name:         name,
		Email:        email,
		Role:         RoleUser,
		GoogleID:     g.ID,
		AvatarURL:    g.Picture,
		AuthProvider: "google",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if s.db.DriverName() == "postgres" {
		query := `
		INSERT INTO users (name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
		`
		var id int64
		err := s.db.QueryRow(query, newUser.Name, newUser.Email, "", string(newUser.Role), "", newUser.GoogleID, newUser.AvatarURL, newUser.AuthProvider, now, now).Scan(&id)
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
	res, err := s.db.Exec(query, newUser.Name, newUser.Email, "", string(newUser.Role), "", newUser.GoogleID, newUser.AvatarURL, newUser.AuthProvider, now, now)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err == nil {
		newUser.ID = id
	}

	return newUser, nil
}

func (s *service) ListUsers(limit, offset int) ([]User, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	err := s.db.QueryRow("SELECT COUNT(1) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
	SELECT id, name, email, password_hash, role, phone, google_id, avatar_url, auth_provider, created_at, updated_at
	FROM users
	ORDER BY id ASC
	LIMIT ? OFFSET ?
	`
	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		var roleStr string
		err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &roleStr, &u.Phone, &u.GoogleID, &u.AvatarURL, &u.AuthProvider, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		u.Role = UserRole(roleStr)
		users = append(users, u)
	}

	return users, total, nil
}
