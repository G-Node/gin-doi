package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/smtp"
	"net/url"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	MAILLOG   = "MailServer"
	DEFAULTTO = "gin@g-node.org" // Fallback email address to notify in case of error
)

func notifyAdmin(dReq *DOIReq, conf *Configuration) error {
	urljoin := func(a, b string) string {
		fallback := fmt.Sprintf("%s/%s (fallback URL join)", a, b)
		base, err := url.Parse(a)
		if err != nil {
			return fallback
		}
		suffix, err := url.Parse(b)
		if err != nil {
			return fallback
		}
		return base.ResolveReference(suffix).String()
	}

	repopath := dReq.Repository
	userlogin := dReq.Username
	useremail := "" // TODO: Change when GOGS sends user email with request
	xmlurl := fmt.Sprintf("%s/%s/doi.xml", conf.Storage.XMLURL, dReq.DOIInfo.UUID)
	uuid := dReq.DOIInfo.UUID
	doitarget := urljoin(conf.Storage.StoreURL, uuid)
	repourl := fmt.Sprintf("%s/%s", conf.GIN.Session.WebAddress(), repopath)

	errorlist := ""
	if len(dReq.ErrorMessages) > 0 {
		errorlist = "The following errors occurred during the dataset preparation\n"
		for idx, msg := range dReq.ErrorMessages {
			errorlist = fmt.Sprintf("%s	%d. %s\n", errorlist, idx+1, msg)
		}
	}

	subject := fmt.Sprintf("New DOI registration request: %s", repopath)

	body := `A new DOI registration request has been received.

	Repository: %s [%s]
	User: %s
	Email address: %s
	DOI XML: %s
	DOI target URL: %s
	UUID: %s

%s
`
	body = fmt.Sprintf(body, repopath, repourl, userlogin, useremail, xmlurl, doitarget, uuid, errorlist)
	return sendMail(subject, body, conf)
}

// sendMail sends an email with a given subject and body.  The supplied
// configuration specifies the server to use, the from address, and a file that
// lists the addresses of the recipients.
func sendMail(subject, body string, conf *Configuration) error {
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

		// Recipient list is read every time a sendMail() is called.
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
		log.Debug("sendMail Done")
	} else {
		log.WithFields(log.Fields{
			"source": MAILLOG,
		}).Infof("Fake mail body: %s", body)
	}
	return nil
}
