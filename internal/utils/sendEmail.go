package utils

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

type MailConfig struct {
	SMTPHost string
	SMTPPort string
	Username string
	Password string
	Sender   string
}

func LoadMailConfig() MailConfig {
	config := MailConfig{
		SMTPHost: os.Getenv("SMTP_HOST"),
		SMTPPort: os.Getenv("SMTP_PORT"),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		Sender:   os.Getenv("SMTP_SENDER"),
	}

	// Debugging: Print loaded configuration
	fmt.Printf("Loaded MailConfig: %+v\n", config)

	return config
}

func SendEmail(config MailConfig, recipient, subject, message string) error {
	auth := smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)

	emailBody := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		config.Sender, recipient, subject, message)

	// Debugging: Print SMTP server address
	smtpAddr := config.SMTPHost + ":" + config.SMTPPort
	fmt.Printf("Attempting to connect to SMTP server: %s\n", smtpAddr)

	err := smtp.SendMail(
		smtpAddr,
		auth,
		config.Sender,
		[]string{recipient},
		[]byte(emailBody),
	)

	if err != nil {
		log.Printf("Failed to send email: %v\n", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Println("Email sent successfully")
	return nil
}
