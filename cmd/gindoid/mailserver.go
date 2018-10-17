package main

import (
	"bytes"
	"net/smtp"

	log "github.com/Sirupsen/logrus"
)

const (
	MAILLOG = "MailServer"
)

type MailServer struct {
	Address   string
	From      string
	DoSend    bool
	Recipient string
}

func (ms *MailServer) SendMail(content string) error {
	if ms.DoSend {
		c, err := smtp.Dial(ms.Address)
		if err != nil {
			log.WithFields(log.Fields{
				"source": MAILLOG,
				"error":  err,
			}).Errorf("Could not reach server")
			return err
		}
		defer c.Close()
		// Set the sender and recipient.
		c.Mail(ms.From)
		c.Rcpt(ms.Recipient)
		// Send the email body.
		wc, err := c.Data()
		if err != nil {
			log.WithFields(log.Fields{
				"source": MAILLOG,
				"error":  err,
			}).Errorf("Could not write Mail")
			return err
		}
		defer wc.Close()
		buf := bytes.NewBufferString(content)
		if _, err = buf.WriteTo(wc); err != nil {
			log.WithFields(log.Fields{
				"source": MAILLOG,
				"error":  err,
			}).Errorf("Could not write Mail")
		}
	} else {
		log.WithFields(log.Fields{
			"source": MAILLOG,
		}).Infof("Fake Mail to: %s, content: %s", ms.Recipient, content)
	}
	return nil
}
