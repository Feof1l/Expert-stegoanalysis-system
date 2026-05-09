package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"main/config"
	"main/logger"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	selectUserQuery = `SELECT user_id, email, name, hashed_password FROM users`
)

func OpenDB(cfg *config.DatabaseConfig, logger *logger.Logger) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", cfg.URI)
	if err != nil {
		if logger != nil {
			logger.Errorf("Failed to open database connection: %v", err)
		}
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if logger != nil {
		logger.Info("Database connection established successfully")
	}
	return db, nil
}

type UserRepository struct {
	db     *sqlx.DB
	logger *logger.Logger
}

func NewUserRepository(db *sqlx.DB, logger *logger.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	var user User
	query := selectUserQuery + ` WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Debugf("User not found by email: %s", email)
			return user, fmt.Errorf("user not found: %w", err)
		}
		r.logger.Errorf("Failed to get user by email %s: %v", email, err)
		return user, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, userID int64) (User, error) {
	var user User
	query := selectUserQuery + ` WHERE user_id = $1`
	err := r.db.GetContext(ctx, &user, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Debugf("User not found by id: %d", userID)
			return user, fmt.Errorf("user not found: %w", err)
		}
		r.logger.Errorf("Failed to get user by id %d: %v", userID, err)
		return user, fmt.Errorf("failed to get user by id: %w", err)
	}
	return user, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, user User) error {
	query := `UPDATE users SET hashed_password = :hashed_password WHERE user_id = :user_id`
	result, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		r.logger.Errorf("Failed to update password for user %d: %v", user.ID, err)
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Errorf("Failed to get rows affected for password update: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Warnf("Password update attempted for non-existent user %d", user.ID)
		return fmt.Errorf("user not found")
	}

	r.logger.Infof("Password updated successfully for user %d", user.ID)
	return nil
}

func (r *UserRepository) Create(ctx context.Context, user User) error {
	query := `INSERT INTO users (email, hashed_password, name) VALUES ($1, $2, $3) RETURNING user_id`
	err := r.db.QueryRowContext(ctx, query, user.Email, user.HashedPassword, user.Name).Scan(&user.ID)
	if err != nil {
		r.logger.Errorf("Failed to create user with email %s: %v", user.Email, err)
		return fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.Infof("User created successfully with id %d and email %s", user.ID, user.Email)
	return nil
}
