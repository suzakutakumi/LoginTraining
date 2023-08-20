package main

import (
	"fmt"
	"net/smtp"
	"strings"
)

type Mail struct {
	Host       string
	Port       int
	SenderMail string
	Password   string
}

func (m Mail) Send(to, subject, body string) error {
	auth := smtp.PlainAuth("", m.SenderMail, m.Password, m.Host)
	recipients := []string{to}
	msg := []byte(strings.ReplaceAll(fmt.Sprintf("To: %s\nSubject: %s\n\n%s", strings.Join(recipients, ","), subject, body), "\n", "\r\n"))
	return smtp.SendMail(fmt.Sprintf("%s:%d", m.Host, m.Port), auth, m.SenderMail, recipients, msg)
}
