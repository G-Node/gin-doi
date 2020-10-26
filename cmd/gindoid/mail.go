package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"

	"github.com/G-Node/gin-cli/ginclient"
	"github.com/gogs/go-gogs-client"
)

const (
	MAILLOG   = "MailServer"
	DEFAULTTO = "gin@g-node.org" // Fallback email address to notify in case of error
)

// notifyAdmin prepares an email notification for new jobs and then calls the
// sendMail function to send it. Also opens an issue on the XMLRepo if set.
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
			errorlist = fmt.Sprintf("%s%d. %s\n", errorlist, idx+1, msg)
		}
	}

	warninglist := ""
	if len(warnings) > 0 {
		warninglist = "The following issues were detected and may need attention\n"
		for idx, msg := range warnings {
			warninglist = fmt.Sprintf("%s%d. %s\n", warninglist, idx+1, msg)
		}
	}

	subject := fmt.Sprintf("New DOI registration request: %s", repopath)

	namestr := username
	if realname != "" {
		namestr = fmt.Sprintf("%s (%s)", namestr, realname)
	}

	body := `A new DOI registration request has been received.

- Repository: %s [%s]
- User: %s
- Email address: %s
- DOI XML: %s
- DOI target URL: %s

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

	issueErr := createIssue(job, body, conf)
	mailErr := sendMail(recipients, subject, body, conf)
	if issueErr != nil && mailErr != nil {
		// both failed; return error to let the user know that the request failed
		// The underlying errors are already logged
		return fmt.Errorf("Failed to notify admins of new request: %s (%s)", job.Metadata.SourceRepository, job.Metadata.Identifier.ID)
	}
	return nil
}

// notifyUser prepares an email notification to the user that successfully
// submitted a request.
func notifyUser(job *RegistrationJob) error {
	doi := job.Metadata.Identifier.ID
	conf := job.Config
	repopath := job.Metadata.SourceRepository
	user := job.Metadata.RequestingUser
	username := user.Username
	realname := user.RealName
	useremail := user.Email
	repourl := fmt.Sprintf("%s/%s", GetGINURL(conf), repopath)

	name := username
	if realname != "" {
		name = realname
	}
	recipients := []string{useremail}

	subject := fmt.Sprintf("DOI registration request: %s", repopath)
	message := fmt.Sprintf(msgSubmitSuccessEmail, name, repourl, doi)

	return sendMail(recipients, subject, message, conf)
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

// createIssue creates a new issue on the configured XMLRepo repository or
// updates an existing one if the title matches.
func createIssue(job *RegistrationJob, content string, conf *Configuration) error {
	repopath := job.Metadata.SourceRepository
	doi := job.Metadata.Identifier.ID
	xmlrepo := job.Config.XMLRepo
	log.Printf("Opening issue on %s", xmlrepo)
	xmldata, _ := job.Metadata.DataCite.Marshal()
	title := fmt.Sprintf("New publication request: %s (%s)", repopath, doi)
	body := fmt.Sprintf("%s\n\n-----\n\nDOI XML:\n\n```xml\n%s\n```", content, xmldata)
	client := job.Config.GIN.Session

	var resp *http.Response
	var posterr error
	if issueID := getIssueID(client, xmlrepo, title); issueID >= 0 {
		path := fmt.Sprintf("api/v1/repos/%s/issues/%d/comments", xmlrepo, issueID)
		data := gogs.CreateIssueCommentOption{Body: body}
		resp, posterr = client.Post(path, data)
	} else {
		path := fmt.Sprintf("api/v1/repos/%s/issues", xmlrepo)
		data := gogs.CreateIssueOption{
			Title: title,
			Body:  body,
		}
		resp, posterr = client.Post(path, data)
	}
	if posterr != nil {
		log.Printf("Failed to create issue or comment on XML repo: %s", posterr.Error())
		return posterr
	} else if resp.StatusCode != http.StatusCreated {
		if msg, err := ioutil.ReadAll(resp.Body); err == nil {
			errmsg := fmt.Sprintf("Failed to create issue or comment on XML repo: [%d] %s", resp.StatusCode, msg)
			log.Printf(errmsg)
			return fmt.Errorf(errmsg)
		} else {
			msg := fmt.Sprintf("Failed to open issue on XML repo: [%d] failed to read response body: %s", resp.StatusCode, err.Error())
			log.Print(msg)
			return fmt.Errorf(msg)
		}
	}
	return nil
}

// getIssueID returns the ID for an issue on a given repo that matches the
// given title. It returns -1 if no issue matching the title is found or an
// error occurs.
func getIssueID(client *ginclient.Client, repo, title string) int64 {
	path := fmt.Sprintf("api/v1/repos/%s/issues", repo)
	resp, err := client.Get(path)
	if err != nil {
		// log the error and return with -1 and a new issue will be created
		log.Printf("Failed to get issues for repository %s: %s", repo, err.Error())
		return -1
	} else if resp.StatusCode != http.StatusOK {
		// log the error and return with -1 and a new issue will be created
		if msg, err := ioutil.ReadAll(resp.Body); err == nil {
			log.Printf("Failed to get issues for repository %s: [%d] %s", repo, resp.StatusCode, msg)
		} else {
			log.Printf("Failed to get issues for repository %s: [%d] failed to read response body: %s", repo, resp.StatusCode, err.Error())
		}
		return -1
	}

	var issues []gogs.Issue
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to get issues for repository %s: %s", repo, err.Error())
		return -1
	}

	if err := json.Unmarshal(content, &issues); err != nil {
		log.Printf("Failed to get issues for repository %s: failed to unmarshal response: %s", repo, err.Error())
		return -1
	}

	for _, issue := range issues {
		if strings.EqualFold(issue.Title, title) {
			log.Printf("Found issue matching %s: %d", title, issue.Index)
			return issue.ID
		}
	}

	return -1
}
