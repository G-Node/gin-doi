package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/smtp"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	MAILLOG   = "MailServer"
	DEFAULTTO = "gin@g-node.org" // Fallback email address to notify in case of error
)

// SendMail sends an email with a given subject and body.  The supplied
// configuration specifies the server to use, the from address, and a file that
// lists the addresses of the recipients.
func SendMail(subject, body string, conf *Configuration) error {
	if conf.Email.Server != "" {
		log.Debug("Preparing mail")
		c, err := smtp.Dial(conf.Email.Server)
		if err != nil {
			log.WithFields(log.Fields{
				"source": MAILLOG,
				"error":  err,
			}).Errorf("Could not reach server")
			return err
		}
		defer c.Close()
		// Set the sender and recipient.
		c.Mail(conf.Email.From)
		message := fmt.Sprintf("From: %s\nSubject: %s", conf.Email.From, subject)

		// Recipient list is read every time a SendMail() is called.
		// This way, the recipient list can be changed without restarting the service.
		emailfile, err := os.Open(conf.Email.RecipientsFile)
		if err == nil {
			defer emailfile.Close()
			filereader := bufio.NewReader(emailfile)
			for address, lerr := filereader.ReadString('\n'); lerr == nil; address, lerr = filereader.ReadString('\n') {
				address = strings.TrimSpace(address)
				log.Debugf("To: %s", address)
				c.Rcpt(address)
				message = fmt.Sprintf("%s\nTo: %s", message, address)
			}
		} else {
			log.Errorf("Email file %s could not be read: %s", conf.Email.RecipientsFile, err.Error())
			log.Errorf("Notifying %s", DEFAULTTO)
			log.Debugf("To: %s", DEFAULTTO)
			c.Rcpt(DEFAULTTO)
			message = fmt.Sprintf("%s\nTo: %s", message, DEFAULTTO)
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
