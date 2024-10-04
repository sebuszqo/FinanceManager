package emailService

import (
	"bytes"
	"fmt"
	"github.com/joho/godotenv"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"path/filepath"
	"sync"
)

const (
	subjectRegistrationConfirmation  = "Confirm your registration"
	templateRegistrationConfirmation = "registration_confirmation.html"
	subjectResetPassword             = "Reset your password"
	templateResetPassword            = "reset_password.html"
	subjectTwoFactorCode             = "Your 2FA code"
	templateTwoFactorCode            = "two_factor_code.html"
)

type EmailData interface {
	TemplateFileName() string
	Subject() string
}

type EmailSender interface {
	QueueEmail(to string, data EmailData)
}

type RegistrationConfirmationData struct {
	UserName string
	Code     string
}

func (r RegistrationConfirmationData) TemplateFileName() string {
	return templateRegistrationConfirmation
}

func (r RegistrationConfirmationData) Subject() string {
	return subjectRegistrationConfirmation
}

type ResetPasswordData struct {
	UserName string
	Code     string
}

func (r ResetPasswordData) TemplateFileName() string {
	return templateResetPassword
}

func (r ResetPasswordData) Subject() string {
	return subjectResetPassword
}

type TwoFactorCodeData struct {
	UserName string
	Code     string
}

func (r TwoFactorCodeData) TemplateFileName() string {
	return templateTwoFactorCode
}

func (r TwoFactorCodeData) Subject() string {
	return subjectTwoFactorCode
}

type EmailService struct {
	from         string
	password     string
	templatesDir string
	smtpHost     string
	smtpPort     string
	taskQueue    chan EmailTask
}

type EmailTask struct {
	to           string
	templateFile string
	data         EmailData
	subject      string
}

var (
	instance *EmailService
	once     sync.Once
)

func NewEmailService() *EmailService {
	once.Do(func() {
		err := godotenv.Load()
		if err != nil {
			log.Fatalf("Error loading .env file")
		}

		templatesDir := os.Getenv("TEMPLATES_DIR")
		if templatesDir == "" {
			log.Fatalf("TEMPLATES_DIR is not set in .env file")
		}

		email := os.Getenv("EMAIL_ADDRESS")
		if email == "" {
			log.Fatalf("EMAIL_ADDRESS is not set in .env file")
		}
		password := os.Getenv("EMAIL_PASSWORD")
		if password == "" {
			log.Fatalf("EMAIL_PASSWORD is not set in .env file")
		}

		instance = &EmailService{
			from:         email,
			password:     password,
			templatesDir: templatesDir,
			smtpHost:     "smtp.gmail.com",
			smtpPort:     "587",
			taskQueue:    make(chan EmailTask, 100),
		}

		go instance.worker()
	})
	return instance
}

func (s *EmailService) worker() {
	for task := range s.taskQueue {
		err := s.sendTemplatedEmail(task.to, task.templateFile, task.data, task.subject)
		if err != nil {
			log.Printf("Error sending email to %s: %v", task.to, err)
		}
	}
}

func (s *EmailService) QueueEmail(to string, data EmailData) {
	s.taskQueue <- EmailTask{to, data.TemplateFileName(), data, data.Subject()}
}

func (s *EmailService) sendTemplatedEmail(to, templateFileName string, data EmailData, subject string) error {
	templatePath := filepath.Join(s.templatesDir, templateFileName)
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template file does not exist: %v", err)
	}

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("error parsing template: %v", err)
	}

	var body bytes.Buffer
	err = tmpl.Execute(&body, data)
	if err != nil {
		return fmt.Errorf("error executing template: %v", err)
	}

	message := []byte("Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\";\r\n\r\n" +
		body.String())

	auth := smtp.PlainAuth("", s.from, s.password, s.smtpHost)
	err = smtp.SendMail(s.smtpHost+":"+s.smtpPort, auth, s.from, []string{to}, message)
	if err != nil {
		return fmt.Errorf("error sending email: %v", err)
	}
	return nil
}
