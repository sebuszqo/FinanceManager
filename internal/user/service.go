package user

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/badoux/checkmail"
	emailService "github.com/sebuszqo/FinanceManager/internal/email"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

const (
	maxEmailLength     = 35
	minEmailLength     = 3
	maxLoginLength     = 30
	minLoginLength     = 5
	bcryptCost         = 12
	defaultCodeTimeout = 2
	CodeVerifyType     = "verify"
	CodeChangeEmail    = "email_upd"
)

var (
	ErrInvalidEmail             = fmt.Errorf("email address is not valid")
	ErrEmailLength              = fmt.Errorf("email address is too long or too shord, max length: %d, min lenght: %d", maxEmailLength, minEmailLength)
	ErrLoginLength              = fmt.Errorf("login it too long, max length: %d, min lenght: %d", maxLoginLength, minLoginLength)
	ErrEmailAlreadyExists       = errors.New("email already exists")
	ErrInternalError            = errors.New("internal Server Error")
	ErrLoginAlreadyExists       = errors.New("login already exists")
	ErrUserAlreadyVerified      = errors.New("user already verified")
	ErrInvalidVerificationCode  = errors.New("invalid verification code")
	ErrVerificationCodeExpired  = errors.New("verification code expired")
	ErrTooManyEmailCodeRequests = errors.New("too many email code requests")
	ErrInvalidOldPassword       = errors.New("invalid old password")
)

type User struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	Login            string    `json:"login"`
	PasswordHash     string    `json:"-"`
	TwoFactorEnabled bool      `json:"two_factor_enabled"`
	TwoFactorMethod  string    `json:"two_factor_method"`
	HashToken        string    `json:"-"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	IsActive         bool      `json:"is_active"`
}

type Service interface {
	Register(email, login, password string) (*User, error)
	VerifyRegistrationCode(userID, code string) error
	GenerateVerificationCode(user *User) error
	GetUserByID(userId string) (*User, error)
	SendVerificationCode(user *User) error
	SaveEmailVerificationCode(userID string, code string, expiresAt time.Time, codeType string) error
	SendEmailChangeVerificationCode(user *User, newEmail string) error
	GetEmailVerificationCode(userID string) (string, string, string, time.Time, time.Time, error)
	GetUserByLoginOrEmail(loginOrEmail string) (*User, error)
	DeleteEmailTwoFactorCode(userID string) error
	ChangePasswordWithOldPassword(userID, oldPassword, newPassword string) error
	changePassword(userID, newPassword string) error
	ResetPassword(userID, newPassword string) error
	RequestEmailChange(userID, newEmail string) error
	ConfirmEmailChange(userID, code string) error
}

type service struct {
	repo         Repository
	emailService emailService.EmailSender
}

func NewUserService(repo Repository, emailService emailService.EmailSender) Service {
	return &service{
		repo:         repo,
		emailService: emailService,
	}
}

func hashPassword(password string) (string, error) {
	var passwordBytes = []byte(password)

	hashedPasswordBytes, err := bcrypt.GenerateFromPassword(passwordBytes, bcryptCost)

	return string(hashedPasswordBytes), err
}

func GenerateVerificationCode() (string, error) {
	code := make([]byte, 6)
	_, err := rand.Read(code)
	if err != nil {
		return "", fmt.Errorf("could not generate verification code: %v", err)
	}
	for i := range code {
		code[i] = '0' + (code[i] % 10)
	}

	return string(code), nil
}

func generateHashToken() (string, error) {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		return "", fmt.Errorf("could not generate hash token: %v", err)
	}
	return hex.EncodeToString(token), nil
}

func validateEmailAddress(email string) error {
	err := checkmail.ValidateFormat(email)
	if err != nil {
		fmt.Println("Email Validation FORMAT check error")
		return ErrInvalidEmail
	}

	err = checkmail.ValidateHost(email)
	if err != nil {
		if !strings.Contains("timeout", err.Error()) {
			fmt.Println("Email Validation HOST check error", err)
			return ErrInvalidEmail
		}
		fmt.Println("Timeout continuing without host check ...")
	}
	if len(email) > maxEmailLength || len(email) <= minEmailLength {
		fmt.Println("Email Validation length check error")
		return ErrEmailLength
	}
	return nil
}

func (s *service) Register(email, login, password string) (*User, error) {
	err := validateEmailAddress(email)
	if err != nil {
		return nil, err
	}

	if len(login) == 0 {
		parts := strings.Split(email, "@")
		if len(parts) < 2 {
			fmt.Println("Email Validation length check error")
			return nil, ErrInvalidEmail
		}
		login = parts[0]
	} else if len(login) > maxLoginLength || len(login) < minLoginLength {
		fmt.Println("Login Validation length check error")
		return nil, ErrLoginLength
	}

	existingUser, err := s.repo.userExistsByLoginOrEmail(login, email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		fmt.Println("Error with database request")
		return nil, ErrInternalError
	}

	if existingUser != nil {
		if existingUser.Login == login {
			fmt.Println("Login already exists")
			return nil, ErrLoginAlreadyExists
		} else if existingUser.Email == email {
			return nil, ErrEmailAlreadyExists
		}
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		fmt.Println("Error during hashing the password")
		return nil, ErrInternalError
	}

	hashToken, err := generateHashToken()
	if err != nil {
		fmt.Println("Error during generating a hashToken")
		return nil, ErrInternalError
	}

	user := &User{
		Email:        email,
		Login:        login,
		PasswordHash: passwordHash,
		HashToken:    hashToken,
	}

	err = s.repo.createUser(user)
	if err != nil {
		fmt.Println("Error during creating the user: ", err)
		return nil, ErrInternalError
	}

	err = s.SendVerificationCode(user)
	if err != nil {
		fmt.Println("Error during sending verification email: ", err)
		return nil, ErrInternalError
	}

	return user, nil
}

func (s *service) SendVerificationCode(user *User) error {
	newCode, err := GenerateVerificationCode()
	if err != nil {
		return fmt.Errorf("could not generate verification code: %v", err)
	}

	expirationTime := time.Now().Add(10 * time.Minute).UTC()
	err = s.repo.saveEmailVerificationCode(user.ID, newCode, expirationTime, CodeVerifyType, "")
	if err != nil {
		fmt.Printf("Error saving verification code: %v\n", err)
		return fmt.Errorf("could not save verification code: %v", err)
	}

	s.emailService.QueueEmail(user.Email, emailService.RegistrationConfirmationData{
		UserName: user.Login,
		Code:     newCode,
	})

	return nil
}

func (s *service) VerifyRegistrationCode(email, code string) error {
	user, err := s.repo.getUserByEmail(email)
	if err != nil {
		fmt.Println("Error getting user from db with ID, ", email, err)
		if errors.Is(err, ErrUserNotFound) {
			return ErrUserNotFound
		}
		return ErrInternalError
	}

	if user.IsActive {
		fmt.Println("user already verified")
		return ErrUserAlreadyVerified
	}

	storedCode, codeType, _, expiryTime, _, err := s.repo.getEmailVerificationCode(user.ID)

	if err != nil {
		fmt.Println("cannot get code from db")
		return ErrInvalidVerificationCode
	}

	if codeType != CodeVerifyType {
		return ErrInvalidVerificationCode
	}

	if storedCode != code {
		fmt.Println("invalid verification code")
		return ErrInvalidVerificationCode
	}

	if time.Now().UTC().After(expiryTime) {
		fmt.Println("invalid verification code - code expired")
		return ErrVerificationCodeExpired
	}

	err = s.repo.updateEmailVerified(user.ID, true)
	if err != nil {
		fmt.Println("issue during updating verified account")
		return ErrInternalError
	}

	err = s.repo.deleteEmailTwoFactorCode(user.ID)
	return nil
}

func (s *service) ReSendVerificationCode(user *User) error {
	_, _, _, _, createdAt, err := s.repo.getEmailVerificationCode(user.ID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return err
		} else {
			return ErrInternalError
		}
	}

	createdAtUTC := createdAt.UTC()
	nowUTC := time.Now().UTC()

	timeSinceLastCode := nowUTC.Sub(createdAtUTC)
	if timeSinceLastCode.Minutes() < defaultCodeTimeout {
		return ErrTooManyEmailCodeRequests
	}

	newCode, err := GenerateVerificationCode()
	if err != nil {
		return fmt.Errorf("could not generate verification code: %v", err)
	}

	expirationTime := time.Now().UTC().Add(10 * time.Minute)

	err = s.repo.saveEmailVerificationCode(user.ID, newCode, expirationTime, CodeVerifyType, "")
	if err != nil {
		return fmt.Errorf("could not save verification code: %v", err)
	}

	s.emailService.QueueEmail(user.Email, emailService.RegistrationConfirmationData{
		UserName: user.Login,
		Code:     newCode,
	})

	return nil
}

func (s *service) ChangePasswordWithOldPassword(userID, oldPassword, newPassword string) error {
	user, err := s.repo.getUserByID(userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrUserNotFound
		} else {
			return ErrInternalError
		}
	}

	if !doPasswordsMatch(user.PasswordHash, oldPassword) {
		return ErrInvalidOldPassword
	}

	return s.changePassword(userID, newPassword)
}

func (s *service) changePassword(userID, newPassword string) error {

	newPasswordHash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("could not hash password: %v", err)
	}

	newHashToken, err := generateHashToken()
	if err != nil {
		return fmt.Errorf("could not generate hash token: %v", err)
	}

	err = s.repo.updateUserPasswordAndHashToken(userID, newPasswordHash, newHashToken)
	if err != nil {
		return fmt.Errorf("could not update user password: %v", err)
	}

	return nil
}

func doPasswordsMatch(hashedPassword, currPassword string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(hashedPassword), []byte(currPassword))
	return err == nil
}

func (s *service) GenerateVerificationCode(user *User) error {
	err := s.ReSendVerificationCode(user)
	if err != nil {
		if errors.Is(err, ErrTooManyEmailCodeRequests) {
			return err
		}
		return ErrInternalError
	}
	return nil
}

func (s *service) SendEmailChangeVerificationCode(user *User, newEmail string) error {
	newCode, err := GenerateVerificationCode()
	if err != nil {
		return fmt.Errorf("could not generate verification code: %v", err)
	}

	expirationTime := time.Now().Add(10 * time.Minute).UTC()
	err = s.repo.saveEmailVerificationCode(user.ID, newCode, expirationTime, CodeChangeEmail, newEmail)
	if err != nil {
		fmt.Printf("Error saving verification code: %v\n", err)
		return fmt.Errorf("could not save verification code: %v", err)
	}

	s.emailService.QueueEmail(newEmail, emailService.RegistrationConfirmationData{
		UserName: user.Login,
		Code:     newCode,
	})

	return nil
}

func (s *service) RequestEmailChange(userID, newEmail string) error {
	_, err := s.repo.getUserByEmail(newEmail)
	if err == nil {
		return ErrEmailAlreadyExists
	} else if !errors.Is(err, ErrUserNotFound) {
		return fmt.Errorf("error checking email existence: %v", err)
	}
	err = validateEmailAddress(newEmail)
	if err != nil {
		return err
	}
	existingUser, err := s.repo.getUserByID(userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrUserNotFound
		} else {
			return ErrInternalError
		}
	}
	err = s.SendEmailChangeVerificationCode(existingUser, newEmail)
	if err != nil {
		fmt.Println("Error during sending verification email: ", err)
		return ErrInternalError
	}
	return nil
}

func (s *service) ConfirmEmailChange(userID, code string) error {
	storedCode, codeType, newEmail, expiryTime, _, err := s.repo.getEmailVerificationCode(userID)
	if err != nil {
		fmt.Println("cannot get code from db")
		return ErrInvalidVerificationCode
	}

	if codeType != CodeChangeEmail {
		return ErrInvalidVerificationCode
	}

	if storedCode != code {
		fmt.Println("invalid verification code")
		return ErrInvalidVerificationCode
	}

	if time.Now().UTC().After(expiryTime) {
		fmt.Println("invalid verification code - code expired")
		return ErrVerificationCodeExpired
	}
	err = s.repo.updateEmail(userID, newEmail)
	if err != nil {
		return ErrInternalError
	}
	_ = s.repo.deleteEmailTwoFactorCode(userID)
	return nil
}

func (s *service) GetUserByID(userID string) (*User, error) {
	return s.repo.getUserByID(userID)
}

func (s *service) SaveEmailVerificationCode(userID string, code string, expiresAt time.Time, codeType string) error {
	return s.repo.saveEmailVerificationCode(userID, code, expiresAt, codeType, "")
}

func (s *service) GetEmailVerificationCode(userID string) (string, string, string, time.Time, time.Time, error) {
	return s.repo.getEmailVerificationCode(userID)
}

func (s *service) GetUserByLoginOrEmail(loginOrEmail string) (*User, error) {
	return s.repo.getUserByLoginOrEmail(loginOrEmail)
}

func (s *service) DeleteEmailTwoFactorCode(userID string) error {
	return s.repo.deleteEmailTwoFactorCode(userID)
}

func (s *service) ResetPassword(userID, newPassword string) error {
	return s.changePassword(userID, newPassword)
}
