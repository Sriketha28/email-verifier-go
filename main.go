package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// isValidEmail checks the syntax of an email address
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// getMXRecords retrieves MX records for the domain
func getMXRecords(domain string) ([]*net.MX, error) {
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return nil, err
	}
	return mxRecords, nil
}

// checkSMTP verifies if the email exists using an SMTP connection
func checkSMTP(email, domain string) bool {
	mxRecords, err := getMXRecords(domain)
	if err != nil || len(mxRecords) == 0 {
		color.Red("❌ No valid mail server found for domain: %s", domain)
		return false
	}

	// Connect to the first mail server
	mx := mxRecords[0].Host
	color.Cyan("🔍 Checking SMTP server: %s", mx)

	conn, err := net.DialTimeout("tcp", mx+":25", 5*time.Second)
	if err != nil {
		color.Red("❌ Failed to connect to mail server: %v", err)
		return false
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, mx)
	if err != nil {
		color.Red("❌ Failed to create SMTP client: %v", err)
		return false
	}
	defer client.Close()

	// Try TLS if supported
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{InsecureSkipVerify: true, ServerName: mx}
		if err = client.StartTLS(tlsConfig); err != nil {
			color.Red("❌ Failed to start TLS: %v", err)
			return false
		}
	}

	// Use a fake sender email
	fakeSender := "verify@example.com"
	if err = client.Mail(fakeSender); err != nil {
		color.Red("❌ MAIL FROM command failed: %v", err)
		return false
	}

	// Check recipient email
	if err = client.Rcpt(email); err != nil {
		color.Red("❌ Email does not exist: %v", err)
		return false
	}

	color.Green("✅ Email exists: %s", email)
	return true
}

// verifyEmail performs syntax, MX record, and SMTP checks
func verifyEmail(email string) {
	if !isValidEmail(email) {
		color.Red("❌ Invalid email format: %s", email)
		return
	}

	// Extract domain
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		color.Red("❌ Invalid email format: %s", email)
		return
	}
	domain := parts[1]

	// Check MX records
	mxRecords, err := getMXRecords(domain)
	if err != nil || len(mxRecords) == 0 {
		color.Red("❌ No valid mail server found for domain: %s", domain)
		return
	}

	color.Green("✔️ Valid email format and domain exists: %s", email)

	// Check if email exists via SMTP
	checkSMTP(email, domain)
}

// processFile reads emails from a file and verifies them
func processFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		color.Red("❌ Failed to open file: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		email := strings.TrimSpace(scanner.Text())
		if email != "" {
			verifyEmail(email)
			fmt.Println()
		}
	}

	if err := scanner.Err(); err != nil {
		color.Red("❌ Error reading file: %v", err)
	}
}

func main() {
	// Command-line arguments
	singleEmail := flag.String("email", "", "Email address to verify")
	filePath := flag.String("file", "", "Path to a file containing emails (one per line)")
	flag.Parse()

	// Ensure input is provided
	if *singleEmail == "" && *filePath == "" {
		color.Yellow("Usage:")
		color.Cyan("  go run main.go -email test@example.com")
		color.Cyan("  go run main.go -file emails.txt")
		os.Exit(1)
	}

	// Verify single email
	if *singleEmail != "" {
		verifyEmail(*singleEmail)
	}

	// Verify emails from file
	if *filePath != "" {
		processFile(*filePath)
	}
}
