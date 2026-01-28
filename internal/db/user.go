package db

import (
	"database/sql"
	"fmt"
	"time"
)

// User 用户模型
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Email        string    `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastLogin    *time.Time `json:"last_login,omitempty"`
}

// UserRepository 用户仓储
type UserRepository struct {
	db *DB
}

// NewUserRepository 创建用户仓储
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByUsername 根据用户名获取用户
func (r *UserRepository) GetByUsername(username string) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(`
		SELECT id, username, password_hash, role, COALESCE(email, ''), created_at, updated_at, last_login
		FROM users WHERE username = ?
	`, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Role,
		&user.Email, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("用户不存在: %s", username)
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetByID 根据 ID 获取用户
func (r *UserRepository) GetByID(id int64) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(`
		SELECT id, username, password_hash, role, COALESCE(email, ''), created_at, updated_at, last_login
		FROM users WHERE id = ?
	`, id).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Role,
		&user.Email, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("用户不存在")
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Create 创建用户
func (r *UserRepository) Create(user *User) error {
	result, err := r.db.Exec(`
		INSERT INTO users (username, password_hash, role, email)
		VALUES (?, ?, ?, ?)
	`, user.Username, user.PasswordHash, user.Role, user.Email)

	if err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}

	id, _ := result.LastInsertId()
	user.ID = id
	return nil
}

// UpdateLastLogin 更新最后登录时间
func (r *UserRepository) UpdateLastLogin(id int64) error {
	_, err := r.db.Exec(`
		UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?
	`, id)
	return err
}

// List 列出所有用户
func (r *UserRepository) List() ([]*User, error) {
	rows, err := r.db.Query(`
		SELECT id, username, role, COALESCE(email, ''), created_at, last_login
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		if err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.Email, &user.CreatedAt, &user.LastLogin); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
