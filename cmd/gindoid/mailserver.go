package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/smtp"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
)

const (
	MAILLOG = "MailServer"
)

type MailServer struct {
	Address   string
	From      string
	DoSend    bool
	EmailList string
}

func (ms *MailServer) SendMail(subject, body string) error {
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

		// Recipient list is read every time a SendMail() is called.
		// This way, the recipient list can be changed without restarting the service.
		emailfile, err := os.Open(ms.EmailList)
		if err != nil {
			log.Errorf("Email file %s could not be read", ms.EmailList)
		}
		filereader := bufio.NewReader(emailfile)
		message := fmt.Sprintf("From: %s\nSubject: %s", ms.From, subject)
		for address, lerr := filereader.ReadString('\n'); lerr == nil; address, lerr = filereader.ReadString('\n') {
			address = strings.TrimSpace(address)
			log.Debugf("To: %s", address)
			c.Rcpt(address)
			message = fmt.Sprintf("%s\nTo: %s", message, address)
		}

		message = fmt.Sprintf("%s\n\n%s", message, body)
		// Send the email body.
		log.Debug("Sending mail")

		wc, err := c.Data()
		if err != nil {
			log.WithFields(log.Fields{
				"source": MAILLOG,
				"error":  err,
			}).Errorf("Could not write mail")
			return err
		}
		defer wc.Close()
		buf := bytes.NewBufferString(message)
		if _, err = buf.WriteTo(wc); err != nil {
			log.WithFields(log.Fields{
				"source": MAILLOG,
				"error":  err,
			}).Errorf("Could not write mail")
		}
		log.Debug("SendMail Done")
	} else {
		log.WithFields(log.Fields{
			"source": MAILLOG,
		}).Infof("Fake mail body: %s", body)
	}
	return nil
}
