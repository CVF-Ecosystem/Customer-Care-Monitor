package notifications

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

type EmailNotifier struct {
	smtpHost string
	smtpPort int
	username string
	password string
	from     string
	to       []string
}

func NewEmailNotifier(smtpHost string, smtpPort int, username, password, from string, to []string) *EmailNotifier {
	return &EmailNotifier{smtpHost: smtpHost, smtpPort: smtpPort, username: username, password: password, from: from, to: to}
}

func (e *EmailNotifier) Send(ctx context.Context, subject, body string) error {
	session, err := openSMTPSession(ctx, e.smtpHost, e.smtpPort)
	if err != nil {
		return err
	}
	defer session.close()
	client := session.client
	if err := client.Hello("localhost"); err != nil {
		return failedDelivery(smtpStage("hello", err))
	}
	if err := smtpStartTLS(client, e.smtpHost); err != nil {
		return err
	}
	auth := smtp.PlainAuth("", e.username, e.password, e.smtpHost)
	if ok, _ := client.Extension("AUTH"); !ok {
		// The pinned smtp.SendMail path permits HELO fallback when EHLO is
		// rejected (smtp.Client then exposes no ESMTP extensions). Keep rejecting
		// a successful EHLO response that omits AUTH.
		if session.ehlo == nil || session.ehlo.accepted() {
			return failedDelivery("auth_unavailable")
		}
	} else if err := client.Auth(auth); err != nil {
		return failedDelivery(smtpStage("auth", err))
	}
	if err := client.Mail(e.from); err != nil {
		return failedDelivery(smtpStage("mail_from", err))
	}
	for _, recipient := range e.to {
		if err := client.Rcpt(recipient); err != nil {
			return failedDelivery(smtpStage("recipient", err))
		}
	}
	data, err := client.Data()
	if err != nil {
		return failedDelivery(smtpStage("data_command", err))
	}
	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n", e.from, strings.Join(e.to, ","), subject)
	if _, err := data.Write([]byte(headers + body)); err != nil {
		_ = data.Close()
		// A partial write can still have placed some or all content on the wire.
		// Closing may also transmit the DATA terminator, so the receiver outcome
		// cannot be proven rejected and must not be automatically retried.
		return unknownDelivery("data_write")
	}
	if err := data.Close(); err != nil {
		return smtpAcceptedDataError(err)
	}
	// The final DATA 250 is the acceptance boundary. QUIT cannot undo it.
	_ = client.Quit()
	return nil
}

func (e *EmailNotifier) HealthCheck(ctx context.Context) error {
	session, err := openSMTPSession(ctx, e.smtpHost, e.smtpPort)
	if err != nil {
		return err
	}
	defer session.close()
	if err := session.client.Hello("localhost"); err != nil {
		return failedDelivery(smtpStage("hello", err))
	}
	return nil
}
