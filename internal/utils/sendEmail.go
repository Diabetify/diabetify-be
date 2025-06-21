package utils

import (
	"crypto/tls"
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
	smtpAddr := config.SMTPHost + ":" + config.SMTPPort
	fmt.Printf("Attempting to connect to SMTP server: %s\n", smtpAddr)

	client, err := smtp.Dial(smtpAddr)
	if err != nil {
		log.Printf("Failed to connect to SMTP server: %v\n", err)
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Upgrade to TLS using STARTTLS
	tlsConfig := &tls.Config{
		ServerName: config.SMTPHost,
		MinVersion: tls.VersionTLS12,
	}
	if err = client.StartTLS(tlsConfig); err != nil {
		log.Printf("Failed to start TLS: %v\n", err)
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	auth := smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)
	if err = client.Auth(auth); err != nil {
		log.Printf("Failed to authenticate: %v\n", err)
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	// Set the sender and recipient
	if err = client.Mail(config.Sender); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	if err = client.Rcpt(recipient); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send the email body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to create mail writer: %w", err)
	}

	emailBody := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		config.Sender, recipient, subject, message)

	_, err = writer.Write([]byte(emailBody))
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close mail writer: %w", err)
	}

	if err = client.Quit(); err != nil {
		log.Printf("Failed to close connection properly: %v\n", err)
	}

	log.Println("Email sent successfully")
	return nil
}
