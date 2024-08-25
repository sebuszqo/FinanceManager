package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type User struct {
	ID               string
	Email            string
	Login            string
	IsActive         bool
	PasswordHash     string
	TwoFactorEnabled bool
	TwoFactorMethod  string
	HashToken        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UserRepository interface {
	EnableTwoFactor(userID, method string) error
	GetTwoFactorSecret(userID string) (string, error)
	SaveTwoFactorSecret(userID string, encryptedSecret string) error
	DisableTwoFactor(userID string) error
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) SaveTwoFactorSecret(userID string, encryptedSecret string) error {
	query := `
        INSERT INTO user_two_factor_secrets (user_id, encrypted_secret, created_at)
        VALUES ($1, $2, NOW())
        ON CONFLICT (user_id) DO UPDATE
        SET encrypted_secret = EXCLUDED.encrypted_secret,
            created_at = NOW()
    `
	_, err := r.db.Exec(query, userID, encryptedSecret)
	if err != nil {
		return ErrInternalError
	}
	return nil
}

func (r *userRepository) GetTwoFactorSecret(userID string) (string, error) {
	var encryptedSecret string
	query := `
        SELECT encrypted_secret
        FROM user_two_factor_secrets
        WHERE user_id = $1
    `
	err := r.db.QueryRow(query, userID).Scan(&encryptedSecret)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrUser2FANotEnabled
		}
		return "", ErrInternalError
	}
	return encryptedSecret, nil
}

func (r *userRepository) EnableTwoFactor(userID, method string) error {
	query := `
		UPDATE users
		SET two_factor_enabled = TRUE,
			two_factor_method = $1,
			updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.db.Exec(query, method, userID)
	if err != nil {
		return ErrInternalError
	}
	return nil
}

// DeleteEmailVerificationCode deletes the verification code for a user after successful verification
func (r *userRepository) DeleteEmailVerificationCode(userID string) error {
	query := `DELETE FROM user_email_verification_codes WHERE user_id = $1`
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("could not delete verification code: %v", err)
	}
	return nil
}

func (r *userRepository) DisableTwoFactor(userID string) error {
	var twoFactorMethod string
	err := r.db.QueryRow(`SELECT two_factor_method FROM users WHERE id = $1`, userID).Scan(&twoFactorMethod)
	if err != nil {
		return fmt.Errorf("could not retrieve two-factor method: %v", err)
	}

	query := `
		UPDATE users
		SET two_factor_enabled = FALSE, two_factor_method = ''
		WHERE id = $1
	`
	_, err = r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("could not disable two-factor authentication in users table: %v", err)
	}

	if twoFactorMethod == google2FAAuthMethod {
		query = `
			DELETE FROM user_two_factor_secrets
			WHERE user_id = $1
		`
		_, err = r.db.Exec(query, userID)
		if err != nil {
			return fmt.Errorf("could not delete TOTP secret from user_two_factor_secrets table: %v", err)
		}
	}

	return nil
}
