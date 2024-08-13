package auth

import (
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"log"
)

type Authenticator struct{}

func (g *Authenticator) GenerateSecret(userID string) (string, string, error) {
	secret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "FinanceManager",
		AccountName: userID,
		Algorithm:   otp.AlgorithmSHA512,
	})
	if err != nil {
		log.Println("Error during totp secret generation: ", err)
		return "", "", ErrInternalError
	}

	secretKey := secret.Secret()
	otpURI := secret.URL()
	return otpURI, secretKey, nil
}

func (g *Authenticator) GenerateCode(secret string) (string, error) {
	// Google Authenticator doesn't require generating code here, so return an empty string
	return "", nil
}

func (g *Authenticator) VerifyCode(secret, code string) (bool, error) {
	valid := totp.Validate(code, secret)
	if !valid {
		return false, ErrInvalid2FACode
	}
	return true, nil
}
