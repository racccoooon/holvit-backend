package jobs

import (
	"context"
	"crypto/tls"
	mail "github.com/xhit/go-simple-mail/v2"
	"holvit/config"
	"holvit/repos"
	"time"
)

type SendMailExecutor struct{}

func (e *SendMailExecutor) Execute(ctx context.Context, details repos.QueuedJobDetails) error {
	d := details.(repos.SendMailJobDetails)

	server := mail.NewSMTPClient()

	server.Host = config.C.MailServer.Host
	server.Port = config.C.MailServer.Port

	server.Username = config.C.MailServer.User
	server.Password = config.C.MailServer.Password

	if config.C.MailServer.StartTls {
		server.Encryption = mail.EncryptionTLS
	}

	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second
	server.TLSConfig = &tls.Config{InsecureSkipVerify: config.C.MailServer.AllowInsecure}

	smtpClient, err := server.Connect()

	if err != nil {
		return err
	}

	email := mail.NewMSG()
	email.SetFrom(config.C.MailServer.From)

	for _, to := range d.To {
		email.AddTo(to)
	}

	for _, cc := range d.Cc {
		email.AddTo(cc)
	}

	for _, bcc := range d.Bcc {
		email.AddTo(bcc)
	}

	email.SetSubject(d.Subject)
	email.SetBody(mail.TextHTML, d.Body)

	if email.Error != nil {
		return email.Error
	}

	err = email.Send(smtpClient)
	if err != nil {
		return err
	}

	return nil
}
