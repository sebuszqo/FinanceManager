package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/sebuszqo/FinanceManager/internal/email"
	"github.com/sebuszqo/FinanceManager/internal/user"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

type EmailOrLogin string

const (
	google2FAAuthMethod = "google_authenticator"
	email2FAAuthMethod  = "email"
	defaultCodeTimeout  = 2
	CodeVerifyType      = "verify"
	Code2FAType         = "2fa"
	CodePassType        = "password"
)

var (
	ErrUserNotFound             = errors.New("user not found")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrInternalError            = errors.New("internal Server Error")
	ErrInvalidTwoFactorMethod   = errors.New("two factor auth method not supported")
	ErrUser2FANotEnabled        = errors.New("two factor auth is not enabled")
	ErrInvalid2FACode           = errors.New("2fa code is invalid")
	ErrUser2FAAlreadyEnabled    = errors.New("2fa auth already enabled")
	ErrInvalidVerificationCode  = errors.New("invalid verification code")
	ErrVerificationCodeExpired  = errors.New("verification code expired")
	ErrUserNotVerified          = errors.New("user has not been verified")
	ErrTooManyEmailCodeRequests = errors.New("too many email code requests")
	InvalidCodeType             = errors.New("invalid code type")
)

// TwoFactorAuthenticator defines the interface for all 2FA methods
type TwoFactorAuthenticator interface {
	GenerateSecret(userID string) (string, string, error) // Generates a secret used by TOTP
	GenerateCode(secret string) (string, error)           // Generates a code (for email and TOTP)
	VerifyCode(secretOrCode, code string) bool            // Verifies the 2FA code
}

type Service interface {
	Login(emailOrLogin, password string) (*user.User, string, string, error)
	VerifyTwoFactor(sessionToken, code string) (*user.User, string, string, error)
	RegisterTwoFactor(userID string, method string) (string, error)
	RefreshAccessToken(refreshToken string) (string, string, error)
	JWTRefreshTokenMiddleware() func(http.Handler) http.Handler
	JWTAccessTokenMiddleware() func(http.Handler) http.Handler
	SendEmailCode(user *user.User, codeType string) error
	VerifyTwoFactorCode(userID, method, code string) error
	DisableTwoFactorAuth(userID, method, verificationCode string) error
	RequestEmail2FACode(userID string) error
	ResetPassword(email, code, newPassword string) error
	RequestPasswordReset(email string) error
}

type service struct {
	repo           UserRepository
	userService    user.Service
	sessionManager SessionManagerInterface
	jwtManager     JWTManagerInterface
	emailService   emailService.EmailSender
	authenticator  Authenticator
}

func NewAuthService(repo UserRepository, userService user.Service, sessionManager SessionManagerInterface, jwtManager JWTManagerInterface, emailService emailService.EmailSender, authenticator Authenticator) Service {
	s := &service{
		repo:           repo,
		userService:    userService,
		sessionManager: sessionManager,
		jwtManager:     jwtManager,
		emailService:   emailService,
		authenticator:  authenticator,
	}

	return s
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

func (s *service) SendEmailCode(user *user.User, codeType string) error {
	_, storedCodeType, _, _, createdAt, err := s.userService.GetEmailVerificationCode(user.ID)
	if storedCodeType != "" {
		createdAtUTC := createdAt.UTC()
		timeSinceLastCode := time.Now().UTC().Sub(createdAtUTC)
		if timeSinceLastCode.Minutes() < defaultCodeTimeout && storedCodeType != CodeVerifyType && codeType != storedCodeType {
			return ErrTooManyEmailCodeRequests
		}
	}

	newCode, err := GenerateVerificationCode()
	if err != nil {
		return fmt.Errorf("could not generate verification code: %v", err)
	}

	expirationTime := time.Now().UTC().Add(10 * time.Minute)
	err = s.userService.SaveEmailVerificationCode(user.ID, newCode, expirationTime, codeType)
	if err != nil {
		return fmt.Errorf("could not save verification code: %v", err)
	}

	switch codeType {
	case Code2FAType:
		s.emailService.QueueEmail(user.Email, emailService.TwoFactorCodeData{
			UserName: user.Login,
			Code:     newCode,
		})
	case CodePassType:
		s.emailService.QueueEmail(user.Email, emailService.ResetPasswordData{
			UserName: user.Login,
			Code:     newCode,
		})
	case CodeVerifyType:
		s.emailService.QueueEmail(user.Email, emailService.RegistrationConfirmationData{
			UserName: user.Login,
			Code:     newCode,
		})
	default:
		fmt.Println("codeType is not supported in email service - email hasn't been sent")
	}

	return nil
}

func (s *service) GenerateVerificationCode(user *user.User) error {
	err := s.SendEmailCode(user, CodeVerifyType)
	if err != nil {
		if errors.Is(err, ErrTooManyEmailCodeRequests) {
			return err
		}
		return ErrInternalError
	}
	return nil
}

func (s *service) Login(emailOrLogin, password string) (*user.User, string, string, error) {
	existingUser, err := s.userService.GetUserByLoginOrEmail(emailOrLogin)
	if err != nil {
		fmt.Println("error when getting user from database: ", err)
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, "", "", ErrInvalidCredentials
		}
		return nil, "", "", ErrInternalError
	}

	if !existingUser.IsActive {
		err := s.SendEmailCode(existingUser, CodeVerifyType)
		if err != nil {
			return nil, "", "", ErrInternalError
		}
		return nil, "", "", ErrUserNotVerified
	}

	if existingUser == nil || !doPasswordsMatch(existingUser.PasswordHash, password) {
		fmt.Println("password do not much or user doesn't exist in database")
		return nil, "", "", ErrInvalidCredentials
	}

	if existingUser.TwoFactorEnabled {
		switch existingUser.TwoFactorMethod {
		case email2FAAuthMethod:
			err = s.SendEmailCode(existingUser, Code2FAType)
			if err != nil {
				fmt.Println("Error during sending verification email: ", err)
				return nil, "", "", ErrInternalError
			}
		case google2FAAuthMethod:
			fmt.Println("User has google 2fa, nothing is needed")
		default:
			return nil, "", "", ErrInvalidTwoFactorMethod
		}
		sessionToken, err := s.sessionManager.GenerateSessionToken(existingUser.ID, defaultSessionTokenDuration)
		if err != nil {
			return nil, "", "", ErrInternalError
		}
		return existingUser, sessionToken, "", nil
	}

	jwtToken, err := s.jwtManager.GenerateAccessJWT(existingUser.ID, defaultJWTDuration)
	if err != nil {
		fmt.Println("error during JWT generation")
		return nil, "", "", ErrInternalError
	}
	refreshToken, err := s.jwtManager.GenerateRefreshJWT(existingUser.ID, existingUser.HashToken, defaultJWTRefreshDuration)
	if err != nil {
		fmt.Println("error during refresh token generation")
		return nil, "", "", ErrInternalError
	}

	return existingUser, jwtToken, refreshToken, nil
}

func (s *service) VerifyTwoFactor(sessionToken, code string) (*user.User, string, string, error) {
	userID, err := s.sessionManager.VerifySessionToken(sessionToken)
	if err != nil {
		return nil, "", "", err
	}
	existingUser, err := s.userService.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, "", "", ErrUserNotFound
		} else {
			return nil, "", "", ErrInternalError
		}
	}
	if !existingUser.TwoFactorEnabled {
		return nil, "", "", ErrUser2FANotEnabled
	}

	var valid bool
	switch existingUser.TwoFactorMethod {
	case email2FAAuthMethod:
		storedCode, codeType, _, expiryTime, _, err := s.userService.GetEmailVerificationCode(userID)
		if codeType != Code2FAType {
			return nil, "", "", InvalidCodeType
		}
		if err != nil {
			return nil, "", "", err
		}
		if storedCode != code {
			fmt.Println("invalid verification code")
			return nil, "", "", ErrInvalidVerificationCode
		}

		if time.Now().After(expiryTime) {
			fmt.Println("invalid verification code - code expired")
			return nil, "", "", ErrVerificationCodeExpired
		}
		valid = true
		err = s.userService.DeleteEmailTwoFactorCode(userID)
		if err != nil {
			return nil, "", "", ErrInternalError
		}
	case google2FAAuthMethod:
		encryptedSecret, err := s.repo.GetTwoFactorSecret(userID)
		if err != nil {
			return nil, "", "", err
		}
		valid = s.authenticator.VerifyCode(encryptedSecret, code)
	default:
		return nil, "", "", ErrInvalidTwoFactorMethod
	}

	if !valid {
		return nil, "", "", ErrInvalid2FACode
	}

	jwtToken, err := s.jwtManager.GenerateAccessJWT(existingUser.ID, defaultJWTDuration)
	if err != nil {
		return nil, "", "", ErrInternalError
	}
	refreshToken, err := s.jwtManager.GenerateRefreshJWT(existingUser.ID, existingUser.HashToken, defaultJWTDuration)
	if err != nil {
		return nil, "", "", ErrInternalError
	}

	return existingUser, jwtToken, refreshToken, nil
}

func (s *service) RegisterTwoFactor(userID string, method string) (string, error) {
	existingUser, err := s.userService.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrUserNotFound
		} else {
			return "", ErrInternalError
		}
	}

	if existingUser.TwoFactorEnabled {
		return "", ErrUser2FAAlreadyEnabled
	}

	switch method {
	case email2FAAuthMethod:
		err := s.SendEmailCode(existingUser, Code2FAType)
		if err != nil {
			fmt.Println("Error during sending verification email: ", err)
			return "", ErrInternalError
		}
	case google2FAAuthMethod:
		otpURI, secret, err := s.authenticator.GenerateSecret(existingUser.Email)
		if err != nil {
			return "", ErrInternalError
		}
		err = s.repo.SaveTwoFactorSecret(userID, secret)
		if err != nil {
			return "", ErrInternalError
		}

		return otpURI, nil
	default:
		return "", ErrInvalidTwoFactorMethod
	}
	return "", err
}

func (s *service) RequestEmail2FACode(userID string) error {
	existingUser, err := s.userService.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrUserNotFound
		} else {
			return ErrInternalError
		}
	}

	if !existingUser.TwoFactorEnabled {
		return ErrUser2FANotEnabled
	}

	if existingUser.TwoFactorMethod != email2FAAuthMethod {
		return ErrInvalidTwoFactorMethod
	}

	err = s.SendEmailCode(existingUser, Code2FAType)
	if err != nil {
		fmt.Println("Error during sending verification email: ", err)
		return ErrInternalError
	}
	return nil
}

func (s *service) DisableTwoFactorAuth(userID, method, verificationCode string) error {
	existingUser, err := s.userService.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if !existingUser.TwoFactorEnabled {
		return ErrUser2FANotEnabled
	}

	if existingUser.TwoFactorMethod != method {
		return ErrInvalidTwoFactorMethod
	}

	switch existingUser.TwoFactorMethod {
	case google2FAAuthMethod:
		secret, err := s.repo.GetTwoFactorSecret(userID)
		if err != nil {
			return ErrInternalError
		}

		valid := s.authenticator.VerifyCode(secret, verificationCode)
		if !valid || err != nil {
			return ErrInvalid2FACode
		}
	case email2FAAuthMethod:
		storedCode, codeType, _, expiryTime, _, err := s.userService.GetEmailVerificationCode(userID)
		if err != nil {
			return err
		}
		if codeType != Code2FAType {
			return InvalidCodeType
		}
		if storedCode != verificationCode {
			fmt.Println("invalid verification code")
			return ErrInvalid2FACode
		}

		if time.Now().After(expiryTime) {
			fmt.Println("invalid verification code - code expired")
			return ErrVerificationCodeExpired
		}

		err = s.userService.DeleteEmailTwoFactorCode(userID)
		if err != nil {
			return ErrInternalError
		}
	default:
		return ErrInvalidTwoFactorMethod
	}

	err = s.repo.DisableTwoFactor(userID)
	if err != nil {
		return ErrInternalError
	}

	return nil
}

func (s *service) VerifyTwoFactorCode(userID, method, code string) error {
	existingUser, err := s.userService.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrUserNotFound
		}
		return ErrInternalError
	}

	if existingUser.TwoFactorEnabled {
		return ErrUser2FAAlreadyEnabled
	}

	switch method {
	case email2FAAuthMethod:
		storedCode, codeType, _, expiryTime, _, err := s.userService.GetEmailVerificationCode(userID)
		if codeType != Code2FAType {
			return InvalidCodeType
		}
		if err != nil {
			return err
		}
		if storedCode != code {
			fmt.Println("invalid verification code")
			return ErrInvalid2FACode
		}

		if time.Now().After(expiryTime) {
			fmt.Println("invalid verification code - code expired")
			return ErrVerificationCodeExpired
		}

		err = s.userService.DeleteEmailTwoFactorCode(userID)
		if err != nil {
			return ErrInternalError
		}

	case google2FAAuthMethod:
		secret, err := s.repo.GetTwoFactorSecret(userID)
		if err != nil {
			if errors.Is(err, ErrUser2FANotEnabled) {
				return ErrInvalidTwoFactorMethod
			}
			return ErrInternalError
		}
		fmt.Println("SECRET?", secret)
		fmt.Println("INVALID?")
		valid := s.authenticator.VerifyCode(secret, code)
		fmt.Println("VALID?", valid)
		if !valid {
			return ErrInvalid2FACode
		}
	default:
		return ErrInvalidTwoFactorMethod
	}

	err = s.repo.EnableTwoFactor(userID, method)
	if err != nil {
		return ErrInternalError
	}

	return nil
}

// RefreshAccessToken requests are already checked in refresh token middleware
func (s *service) RefreshAccessToken(userID string) (string, string, error) {
	existingUser, err := s.userService.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", "", ErrUserNotFound
		} else {
			return "", "", ErrInternalError
		}
	}
	jwtToken, err := s.jwtManager.GenerateAccessJWT(userID, defaultJWTDuration)
	if err != nil {
		return "", "", ErrInternalError
	}

	newRefreshToken, err := s.jwtManager.GenerateRefreshJWT(userID, existingUser.HashToken, defaultJWTRefreshDuration)
	if err != nil {
		return "", "", ErrInternalError
	}

	return jwtToken, newRefreshToken, nil
}

func (s *service) RequestPasswordReset(email string) error {
	existingUser, err := s.userService.GetUserByLoginOrEmail(email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrUserNotFound
		}
		return ErrInternalError
	}
	if existingUser.TwoFactorEnabled {
		switch existingUser.TwoFactorMethod {
		case google2FAAuthMethod:
			fmt.Println("google 2fa enabled - there is no need to generate new code")
			return nil
		case email2FAAuthMethod:
			err = s.SendEmailCode(existingUser, Code2FAType)
			if err != nil {
				if errors.Is(err, ErrTooManyEmailCodeRequests) {
					return ErrTooManyEmailCodeRequests
				}
				fmt.Println("Error during sending verification email: ", err)
				return ErrInternalError
			}
		default:
			return ErrInvalidTwoFactorMethod
		}
	}
	err = s.SendEmailCode(existingUser, CodePassType)
	if err != nil {
		if errors.Is(err, ErrTooManyEmailCodeRequests) {
			return ErrTooManyEmailCodeRequests
		}
		fmt.Println("Error during sending password reset email: ", err)
		return ErrInternalError
	}
	return nil
}

func (s *service) ResetPassword(email, verificationCode, newPassword string) error {
	existingUser, err := s.userService.GetUserByLoginOrEmail(email)
	if err != nil {
		return ErrUserNotFound
	}

	if existingUser.TwoFactorEnabled {
		switch existingUser.TwoFactorMethod {
		case google2FAAuthMethod:
			secret, err := s.repo.GetTwoFactorSecret(existingUser.ID)
			if err != nil {
				return ErrInternalError
			}

			valid := s.authenticator.VerifyCode(secret, verificationCode)
			if !valid || err != nil {
				return ErrInvalid2FACode
			}
		case email2FAAuthMethod:
			storedCode, codeType, _, expiryTime, _, err := s.userService.GetEmailVerificationCode(existingUser.ID)
			if err != nil {
				if errors.Is(err, user.ErrNoTwoFactorCodeGenerated) {
					return user.ErrNoTwoFactorCodeGenerated
				}
				return ErrInternalError
			}
			if codeType != Code2FAType {
				return InvalidCodeType
			}
			if storedCode != verificationCode {
				return ErrInvalid2FACode
			}

			if time.Now().After(expiryTime) {
				return ErrVerificationCodeExpired
			}

			err = s.userService.DeleteEmailTwoFactorCode(existingUser.ID)
			if err != nil {
				return ErrInternalError
			}
		default:
			return ErrInvalidTwoFactorMethod
		}
	} else {
		storedCode, codeType, _, expiryTime, _, err := s.userService.GetEmailVerificationCode(existingUser.ID)
		if err != nil {
			if errors.Is(err, user.ErrNoTwoFactorCodeGenerated) {
				return user.ErrNoTwoFactorCodeGenerated
			}
			return ErrInternalError
		}
		if codeType != CodePassType {
			return InvalidCodeType
		}
		if storedCode != verificationCode {
			fmt.Println("invalid verification code")
			return ErrInvalid2FACode
		}

		if time.Now().After(expiryTime) {
			fmt.Println("invalid verification code - code expired")
			return ErrVerificationCodeExpired
		}

		err = s.userService.DeleteEmailTwoFactorCode(existingUser.ID)
		if err != nil {
			return ErrInternalError
		}
	}

	err = s.userService.ResetPassword(existingUser.ID, newPassword)
	if err != nil {
		return ErrInternalError
	}
	return nil
}

func doPasswordsMatch(hashedPassword, currPassword string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(hashedPassword), []byte(currPassword))
	return err == nil
}
