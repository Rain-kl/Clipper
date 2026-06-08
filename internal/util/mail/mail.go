/*
Copyright 2026 linux.do
Modified by Arctel.net, 2026

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mail

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"time"
)

// Config represents SMTP mail configuration
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
}

// SendMail sends an HTML email using the provided config and message details
func SendMail(cfg Config, to string, subject, body string) error {
	return SendMailHTML(cfg, to, subject, body)
}

// SendMailHTML sends an HTML format email
func SendMailHTML(cfg Config, to string, subject, body string) error {
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))

	// Header & MIME settings for HTML email
	header := make(map[string]string)
	header["From"] = cfg.Username
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	// If using SSL port 465, we connection via TLS dial
	if cfg.Port == 465 {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         cfg.Host,
		}
		dialer := &net.Dialer{Timeout: 5 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf(errDialTLSFailed, err)
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return fmt.Errorf(errSMTPClientCreationFailed, err)
		}
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			return fmt.Errorf(errSMTPAuthFailed, err)
		}

		if err = client.Mail(cfg.Username); err != nil {
			return fmt.Errorf(errSMTPMailCommandFailed, err)
		}

		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf(errSMTPRcptCommandFailed, err)
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf(errSMTPDataCommandFailed, err)
		}
		defer w.Close()

		_, err = w.Write([]byte(message))
		if err != nil {
			return fmt.Errorf(errSMTPWritingBodyFailed, err)
		}

		return nil
	}

	// For standard port (587 / 25), use smtp.SendMail directly (handles STARTTLS automatically if server supports it)
	err := smtp.SendMail(addr, auth, cfg.Username, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf(errSendMailFailed, err)
	}

	return nil
}

// SendMailWithLog sends a test email and records a detailed SMTP connection log
func SendMailWithLog(cfg Config, to string, subject, body string) (string, error) {
	var logBuf bytes.Buffer
	logLine := func(dir string, format string, args ...interface{}) {
		logBuf.WriteString(fmt.Sprintf("[%s] %s\n", dir, fmt.Sprintf(format, args...)))
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	logLine("System", "Connecting to %s...", addr)

	var conn net.Conn
	var err error
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	if cfg.Port == 465 {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         cfg.Host,
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		logLine("Error", "Connection failed: %v", err)
		return logBuf.String(), err
	}
	defer conn.Close()
	logLine("System", "Connected successfully.")

	// Set a 10-second session deadline for read/write operations
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		logLine("Error", "SMTP client handshake failed: %v", err)
		return logBuf.String(), err
	}
	defer client.Close()

	// If not 465, support STARTTLS if available
	if cfg.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			logLine("C", "STARTTLS")
			tlsConfig := &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         cfg.Host,
			}
			if err = client.StartTLS(tlsConfig); err != nil {
				logLine("Error", "STARTTLS failed: %v", err)
				return logBuf.String(), err
			}
			logLine("S", "220 Ready to start TLS")
		}
	}

	// Authentication
	if cfg.Username != "" && cfg.Password != "" {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		logLine("C", "AUTH PLAIN **********")
		if err = client.Auth(auth); err != nil {
			logLine("Error", "Authentication failed: %v", err)
			return logBuf.String(), err
		}
		logLine("S", "235 Authentication successful")
	}

	// Mail command
	logLine("C", "MAIL FROM:<%s>", cfg.Username)
	if err = client.Mail(cfg.Username); err != nil {
		logLine("Error", "MAIL FROM command failed: %v", err)
		return logBuf.String(), err
	}
	logLine("S", "250 OK")

	// Rcpt command
	logLine("C", "RCPT TO:<%s>", to)
	if err = client.Rcpt(to); err != nil {
		logLine("Error", "RCPT TO command failed: %v", err)
		return logBuf.String(), err
	}
	logLine("S", "250 OK")

	// Data command
	logLine("C", "DATA")
	w, err := client.Data()
	if err != nil {
		logLine("Error", "DATA command failed: %v", err)
		return logBuf.String(), err
	}
	logLine("S", "354 Start mail input")

	// Header & MIME settings for HTML email
	header := make(map[string]string)
	header["From"] = cfg.Username
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	logLine("System", "Sending message body...")
	if _, err = w.Write([]byte(message)); err != nil {
		w.Close()
		logLine("Error", "Writing message body failed: %v", err)
		return logBuf.String(), err
	}
	w.Close()
	logLine("S", "250 OK")

	logLine("C", "QUIT")
	_ = client.Quit()
	logLine("System", "Mail sent successfully!")

	return logBuf.String(), nil
}
