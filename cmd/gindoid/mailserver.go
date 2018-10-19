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

func (ms *MailServer) SendMail(body string) error {
	if ms.DoSend {
		log.Debug("Preparing mail")
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
		log.Debug("Sending mail")
		wc, err := c.Data()
		if err != nil {
			log.WithFields(log.Fields{
				"source": MAILLOG,
				"error":  err,
			}).Errorf("Could not write Mail")
			return err
		}
		defer wc.Close()
		buf := bytes.NewBufferString(body)
		if _, err = buf.WriteTo(wc); err != nil {
			log.WithFields(log.Fields{
				"source": MAILLOG,
				"error":  err,
			}).Errorf("Could not write Mail")
		}
		log.Debug("SendMail Done")
	} else {
		log.WithFields(log.Fields{
			"source": MAILLOG,
		}).Infof("Fake Mail to: %s, body: %s", ms.Recipient, body)
	}
	return nil
}
