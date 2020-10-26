package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/smtp"
	"net/url"
	"os"
	"strings"
)

const (
	MAILLOG   = "MailServer"
	DEFAULTTO = "gin@g-node.org" // Fallback email address to notify in case of error
)

// notifyAdmin prepares an email notification for new jobs and then calls the
// sendMail function to send it.
func notifyAdmin(job *RegistrationJob, errors, warnings []string) error {
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

	doi := job.Metadata.Identifier.ID

	conf := job.Config
	repopath := job.Metadata.SourceRepository
	user := job.Metadata.RequestingUser
	username := user.Username
	realname := user.RealName
	useremail := user.Email
	xmlurl := fmt.Sprintf("%s/%s/doi.xml", conf.Storage.XMLURL, doi)
	doitarget := urljoin(conf.Storage.StoreURL, doi)
	repourl := fmt.Sprintf("%s/%s", GetGINURL(conf), repopath)

	errorlist := ""
	if len(errors) > 0 {
		errorlist = "The following errors occurred during the dataset preparation\n"
		for idx, msg := range errors {
			errorlist = fmt.Sprintf("%s	%d. %s\n", errorlist, idx+1, msg)
		}
	}

	warninglist := ""
	if len(warnings) > 0 {
		warninglist = "The following issues were detected and may need attention\n"
		for idx, msg := range warnings {
			warninglist = fmt.Sprintf("%s	%d. %s\n", warninglist, idx+1, msg)
		}
	}

	subject := fmt.Sprintf("New DOI registration request: %s", repopath)

	namestr := username
	if realname != "" {
		namestr = fmt.Sprintf("%s (%s)", namestr, realname)
	}

	body := `A new DOI registration request has been received.

	Repository: %s [%s]
	User: %s
	Email address: %s
	DOI XML: %s
	DOI target URL: %s

%s

%s
`
	body = fmt.Sprintf(body, repopath, repourl, namestr, useremail, xmlurl, doitarget, errorlist, warninglist)

	recipients := make([]string, 0)
	// Recipient list is read every time a sendMail() is called.
	// This way, the recipient list can be changed without restarting the service.
	emailfile, err := os.Open(conf.Email.RecipientsFile)
	if err == nil {
		defer emailfile.Close()
		filereader := bufio.NewReader(emailfile)
		for address, lerr := filereader.ReadString('\n'); lerr == nil; address, lerr = filereader.ReadString('\n') {
			address = strings.TrimSpace(address)
			recipients = append(recipients, address)
		}
	} else {
		log.Printf("Email file %s could not be read: %s", conf.Email.RecipientsFile, err.Error())
		log.Printf("Notifying %s", DEFAULTTO)
		recipients = []string{DEFAULTTO}
	}
	return sendMail(recipients, subject, body, conf)
}

// sendMail sends an email with a given subject and body. The supplied
// configuration specifies the server to use, the from address, and a file that
// lists the addresses of the recipients.
func sendMail(to []string, subject, body string, conf *Configuration) error {
	if conf.Email.Server == "" {
		log.Printf("Fake mail body: %s", body)
		return nil
	}
	log.Print("Preparing mail")
	c, err := smtp.Dial(conf.Email.Server)
	if err != nil {
		log.Print("Could not reach server")
		return err
	}
	defer c.Close()
	// Set the sender and recipient.
	c.Mail(conf.Email.From)
	message := fmt.Sprintf("From: %s\nSubject: %s", conf.Email.From, subject)
	if to != nil && len(to) > 0 {
		for _, address := range to {
			address = strings.TrimSpace(address)
			log.Printf("To: %s", address)
			c.Rcpt(address)
			message = fmt.Sprintf("%s\nTo: %s", message, address)
		}
	} else {
		log.Print("Potential error: Mail server configured but no recipients specified.")
		log.Printf("Notifying %q", DEFAULTTO)
		c.Rcpt(DEFAULTTO)
		message = fmt.Sprintf("%s\nTo: %s", message, DEFAULTTO)
		body = fmt.Sprintf("Potential error: The following message had no specified recipients\n\n%s", body)
	}

	message = fmt.Sprintf("%s\n\n%s", message, body)
	// Send the email body.
	log.Print("Sending mail")

	wc, err := c.Data()
	if err != nil {
		log.Print("Could not write mail")
		return err
	}
	defer wc.Close()
	buf := bytes.NewBufferString(message)
	if _, err = buf.WriteTo(wc); err != nil {
		log.Print("Could not write mail")
	}
	log.Print("sendMail Done")
	return nil
}
