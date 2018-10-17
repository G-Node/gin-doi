package main

import (
	"bytes"
	"net/smtp"

	log "github.com/Sirupsen/logrus"
)

var (
	MAILLOG = "MailServer"
)

type MailServer struct {
	Adress string
	From   string
	DoSend bool
	Master string
}

func (ms *MailServer) SendMail(content string) error {
	if ms.DoSend {
		c, err := smtp.Dial(ms.Adress)
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
		c.Rcpt(ms.Master)
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
		}).Infof("Fake Mail to: %s, content: %s, Auth:%+v", ms.Master, content)
	}
	return nil
}
