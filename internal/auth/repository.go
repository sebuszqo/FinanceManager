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
	PasswordHash     string
	Salt             string
	TwoFactorEnabled bool
	TwoFactorMethod  string
	TwoFactorSecret  string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UserRepository interface {
	CreateUser(user *User) error
	GetUserByEmail(email string) (*User, error)
	GetUserByLogin(login string) (*User, error)
	UserExistsByLoginOrEmail(login, email string) (*User, error)
	EnableTwoFactor(userID, method string) error
	GetUserByLoginOrEmail(loginOrEmail string) (*User, error)
	GetUserByID(id string) (*User, error)
	GetTwoFactorSecret(userID string) (string, error)
	SaveTwoFactorSecret(userID string, encryptedSecret string) error
	SaveEmailTwoFactorCode(userID, code string, expiresAt time.Time) error
	GetEmailTwoFactorCode(userID string) (string, error)
	DeleteEmailTwoFactorCode(userID string) error
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) CreateUser(user *User) error {
	query := `
		INSERT INTO users (email, login, password_hash, salt, two_factor_enabled, two_factor_method, two_factor_secret, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id;
	`
	var id string
	err := r.db.QueryRow(query, user.Email, user.Login, user.PasswordHash, user.Salt, user.TwoFactorEnabled, user.TwoFactorMethod, user.TwoFactorSecret).Scan(&id)
	if err != nil {
		return fmt.Errorf("could not create user: %v", err)
	}

	user.ID = id
	return nil
}

func (r *userRepository) GetUserByEmail(email string) (*User, error) {
	query := `
		SELECT id, email, login, password_hash, salt, two_factor_enabled, two_factor_method, two_factor_secret, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user User
	err := r.db.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Login, &user.PasswordHash, &user.Salt, &user.TwoFactorEnabled, &user.TwoFactorMethod, &user.TwoFactorSecret, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("could not find user: %v", err)
	}

	return &user, nil
}

func (r *userRepository) GetUserByLogin(login string) (*User, error) {
	query := `
		SELECT id, email, login, password_hash, salt, two_factor_enabled, two_factor_method, two_factor_secret, created_at, updated_at
		FROM users
		WHERE login = $1
	`

	var user User
	err := r.db.QueryRow(query, login).Scan(&user.ID, &user.Email, &user.Login, &user.PasswordHash, &user.Salt, &user.TwoFactorEnabled, &user.TwoFactorMethod, &user.TwoFactorSecret, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("could not find user: %v", err)
	}

	return &user, nil
}

func (r *userRepository) UserExistsByLoginOrEmail(login, email string) (*User, error) {
	query := `
		SELECT id, email, login, password_hash, salt, two_factor_enabled, two_factor_method, two_factor_secret, created_at, updated_at
		FROM users
		WHERE login = $1 OR email = $2
	`

	var user User
	err := r.db.QueryRow(query, login, email).Scan(&user.ID, &user.Email, &user.Login, &user.PasswordHash, &user.Salt, &user.TwoFactorEnabled, &user.TwoFactorMethod, &user.TwoFactorSecret, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("could not find user: %v", err)
	}

	return &user, nil
}

func (r *userRepository) GetUserByLoginOrEmail(loginOrEmail string) (*User, error) {
	query := `
		SELECT id, email, login, password_hash, salt, two_factor_enabled, two_factor_method, two_factor_secret, created_at, updated_at
		FROM users
		WHERE login = $1 OR email = $1
	`

	var user User
	err := r.db.QueryRow(query, loginOrEmail).Scan(&user.ID, &user.Email, &user.Login, &user.PasswordHash, &user.Salt, &user.TwoFactorEnabled, &user.TwoFactorMethod, &user.TwoFactorSecret, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("could not find user: %v", err)
	}

	return &user, nil
}

func (r *userRepository) GetUserByID(id string) (*User, error) {
	query := `
		SELECT id, email, login, password_hash, salt, two_factor_enabled, two_factor_method, two_factor_secret, created_at, updated_at
		FROM users
		WHERE login = $1
	`

	var user User
	err := r.db.QueryRow(query, id).Scan(&user.ID, &user.Email, &user.Login, &user.PasswordHash, &user.Salt, &user.TwoFactorEnabled, &user.TwoFactorMethod, &user.TwoFactorSecret, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("could not find user: %v", err)
	}

	return &user, nil
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

func (r *userRepository) SaveEmailTwoFactorCode(userID, code string, expiresAt time.Time) error {
	query := `
        INSERT INTO user_email_two_factor_codes (user_id, code, created_at, expires_at)
        VALUES ($1, $2, NOW(), $3)
        ON CONFLICT (user_id) DO UPDATE
        SET code = EXCLUDED.code,
            created_at = NOW(),
            expires_at = EXCLUDED.expires_at
    `
	_, err := r.db.Exec(query, userID, code, expiresAt)
	if err != nil {
		return ErrInternalError
	}
	return nil
}

func (r *userRepository) GetEmailTwoFactorCode(userID string) (string, error) {
	var code string
	query := `
        SELECT code
        FROM user_email_two_factor_codes
        WHERE user_id = $1 AND expires_at > NOW()
    `
	err := r.db.QueryRow(query, userID).Scan(&code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("no valid email two-factor code found for user: %v", userID)
		}
		return "", ErrInternalError
	}
	return code, nil
}

func (r *userRepository) DeleteEmailTwoFactorCode(userID string) error {
	query := `
        DELETE FROM user_email_two_factor_codes 
        WHERE user_id = $1
    `
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("could not delete email two-factor code: %v", err)
	}
	return nil
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
