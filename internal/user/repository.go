package user

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrUserNotFound             = errors.New("user not found")
	ErrNoTwoFactorCodeGenerated = errors.New("no two-factor authentication code generated")
)

type Repository interface {
	createUser(user *User) error
	getUserByEmail(email string) (*User, error)
	getUserByLogin(login string) (*User, error)
	userExistsByLoginOrEmail(login, email string) (*User, error)
	getUserByLoginOrEmail(loginOrEmail string) (*User, error)
	getUserByID(id string) (*User, error)
	saveEmailVerificationCode(userID string, code string, expiresAt time.Time, codeType string) error
	updateEmailVerified(userID string, verified bool) error
	getEmailVerificationCode(userID string) (string, string, time.Time, time.Time, error)
	deleteEmailTwoFactorCode(userID string) error
	updateUserPasswordAndHashToken(userID, newPasswordHash, newHashToken string) error
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) Repository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) createUser(user *User) error {
	query := `
		INSERT INTO users (email, login, password_hash, two_factor_enabled, two_factor_method, hash_token, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id;
	`
	var id string
	err := r.db.QueryRow(query, user.Email, user.Login, user.PasswordHash, user.TwoFactorEnabled, user.TwoFactorMethod, user.HashToken).Scan(&id)
	if err != nil {
		return fmt.Errorf("could not create user: %v", err)
	}

	user.ID = id
	return nil
}

func (r *userRepository) getUserByEmail(email string) (*User, error) {
	query := `
		SELECT id, email, login, password_hash, two_factor_enabled, is_verified, two_factor_method, hash_token, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user User
	err := r.db.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Login, &user.PasswordHash, &user.TwoFactorEnabled, &user.IsActive, &user.TwoFactorMethod, &user.HashToken, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("could not find user: %v", err)
	}

	return &user, nil
}

func (r *userRepository) getUserByLogin(login string) (*User, error) {
	query := `
		SELECT id, email, login, password_hash, two_factor_enabled, is_verified, two_factor_method, hash_token, created_at, updated_at
		FROM users
		WHERE login = $1
	`

	var user User
	err := r.db.QueryRow(query, login).Scan(&user.ID, &user.Email, &user.Login, &user.PasswordHash, &user.TwoFactorEnabled, &user.IsActive, &user.TwoFactorMethod, &user.HashToken, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("could not find user: %v", err)
	}

	return &user, nil
}

func (r *userRepository) userExistsByLoginOrEmail(login, email string) (*User, error) {
	query := `
		SELECT id, email, login, password_hash, is_verified, two_factor_enabled, two_factor_method, hash_token, created_at, updated_at
		FROM users
		WHERE login = $1 OR email = $2
	`

	var user User
	err := r.db.QueryRow(query, login, email).Scan(&user.ID, &user.Email, &user.Login, &user.PasswordHash, &user.IsActive, &user.TwoFactorEnabled, &user.TwoFactorMethod, &user.HashToken, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("could not find user: %v", err)
	}

	return &user, nil
}

func (r *userRepository) getUserByLoginOrEmail(loginOrEmail string) (*User, error) {
	query := `
		SELECT id, email, login, password_hash, is_verified, two_factor_enabled, two_factor_method, hash_token, created_at, updated_at
		FROM users
		WHERE login = $1 OR email = $1
	`

	var user User
	err := r.db.QueryRow(query, loginOrEmail).Scan(&user.ID, &user.Email, &user.Login, &user.PasswordHash, &user.IsActive, &user.TwoFactorEnabled, &user.TwoFactorMethod, &user.HashToken, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("could not find user: %v", err)
	}

	return &user, nil
}

func (r *userRepository) getUserByID(id string) (*User, error) {
	query := `
		SELECT id, email, login, password_hash, is_verified, two_factor_enabled, two_factor_method, hash_token, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := r.db.QueryRow(query, id).Scan(&user.ID, &user.Email, &user.Login, &user.PasswordHash, &user.IsActive, &user.TwoFactorEnabled, &user.TwoFactorMethod, &user.HashToken, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("could not find user: %v", err)
	}

	return &user, nil
}

func (r *userRepository) saveEmailVerificationCode(userID string, code string, expiresAt time.Time, codeType string) error {
	query := `
        INSERT INTO user_email_verification_codes (user_id, code, expires_at, type)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (user_id) DO UPDATE
        SET code = $2, expires_at = $3, created_at = CURRENT_TIMESTAMP
    `
	_, err := r.db.Exec(query, userID, code, expiresAt, codeType)
	if err != nil {
		return fmt.Errorf("could not save verification code: %v", err)
	}
	return nil
}

func (r *userRepository) updateEmailVerified(userID string, verified bool) error {
	query := `
        UPDATE users
        SET is_verified = $2
        WHERE id = $1
    `
	_, err := r.db.Exec(query, userID, verified)
	if err != nil {
		return fmt.Errorf("could not update email verification status: %v", err)
	}
	return nil
}

func (r *userRepository) getEmailVerificationCode(userID string) (string, string, time.Time, time.Time, error) {
	query := `
        SELECT code, expires_at, created_at, type
        FROM user_email_verification_codes
        WHERE user_id = $1
    `

	var code string
	var codeType string
	var expiresAt time.Time
	var createdAt time.Time
	err := r.db.QueryRow(query, userID).Scan(&code, &expiresAt, &createdAt, &codeType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", time.Time{}, time.Time{}, ErrNoTwoFactorCodeGenerated
		}
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("could not retrieve verification code: %v", err)
	}

	return code, codeType, expiresAt, createdAt, nil
}

func (r *userRepository) deleteEmailTwoFactorCode(userID string) error {
	query := `
        DELETE FROM user_email_verification_codes 
        WHERE user_id = $1
    `
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("could not delete email two-factor code: %v", err)
	}
	return nil
}

func (r *userRepository) updateUserPasswordAndHashToken(userID, newPasswordHash, newHashToken string) error {
	query := `
        UPDATE users
        SET password_hash = $1,
            hash_token = $2,
            updated_at = $3
        WHERE id = $4
    `
	_, err := r.db.Exec(query, newPasswordHash, newHashToken, time.Now(), userID)
	return err
}
