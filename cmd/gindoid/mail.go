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
	"path"
	"strings"

	"github.com/G-Node/gin-cli/ginclient"
	"github.com/gogs/go-gogs-client"
)

const (
	// MAILLOG is currently not used in any project and should be considered deprecated.
	MAILLOG = "MailServer"
	// DEFAULTTO is a fallback email address to notify in case of error.
	DEFAULTTO = "gin@g-node.org"
)

// notifyAdmin prepares an email notification for new jobs and then calls the
// sendMail function to send it. Also opens an issue on the XMLRepo if set.
// If fullinfo is 'false', only errors and warnings are sent in the
// notification.
func notifyAdmin(job *RegistrationJob, errors, warnings []string, fullinfo bool) error {
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

	subject := fmt.Sprintf("New DOI registration request: %s", repopath)

	namestr := username
	if realname != "" {
		namestr = fmt.Sprintf("%s (%s)", namestr, realname)
	}

	body := ""
	if fullinfo {
		infofmt := `A new DOI registration request has been received.

- Repository: %s [%s]
- User: %s
- Email address: %s
- DOI XML: %s
- DOI target URL: %s
`
		body = fmt.Sprintf(infofmt, repopath, repourl, namestr, useremail, xmlurl, doitarget)
	}

	errorlist := ""
	if len(errors) > 0 {
		errorlist = "\n\nThe following errors occurred during the dataset preparation\n"
		for idx, msg := range errors {
			errorlist = fmt.Sprintf("%s%d. %s\n", errorlist, idx+1, msg)
		}
	}

	warninglist := ""
	if len(warnings) > 0 {
		warninglist = "\n\nThe following issues were detected and may need attention\n"
		for idx, msg := range warnings {
			warninglist = fmt.Sprintf("%s%d. %s\n", warninglist, idx+1, msg)
		}
	}

	// the full info is only requested as the initial notification email.
	// If it is not the initial notification and there are no errors or warnings,
	// send a notification that the DOI has been prepared without issues.
	if !fullinfo && len(errors)+len(warnings) == 0 {
		body = "The repository cloning and zip creation have finished, no issues have been identified\n"
	}

	body = fmt.Sprintf("%s%s%s", body, errorlist, warninglist)

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

	xmldata, _ := job.Metadata.DataCite.Marshal()
	issueContent := body
	if fullinfo {
		// include xml file content
		issueContent = fmt.Sprintf("%s\n\n-----\n\nDOI XML:\n\n```xml\n%s\n```", body, xmldata)
	}
	issueIndex, issueErr := createIssue(job, issueContent, conf)
	issueURL, _ := url.Parse(GetGINURL(conf))
	issueURL.Path = path.Join(conf.XMLRepo, "issues", fmt.Sprintf("%d", issueIndex))
	if issueErr == nil {
		body = fmt.Sprintf("%s\n\nVisit %s for comments and updates on the request.", body, issueURL.String())
	} else {
		body = fmt.Sprintf("%s\n\n%s", body, issueErr.Error())
	}
	mailErr := sendMail(recipients, subject, body, conf)
	if issueErr != nil && mailErr != nil {
		// both failed; return error to let the user know that the request failed
		// The underlying errors are already logged
		return fmt.Errorf("failed to notify admins of new request: %s (%s)", job.Metadata.SourceRepository, job.Metadata.Identifier.ID)
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
	err = c.Mail(conf.Email.From)
	if err != nil {
		// Missing sender is not too bad, log but carry on.
		log.Printf("Error: Could not add mail sender: %q", err.Error())
	}

	message := fmt.Sprintf("From: %s\nSubject: %s", conf.Email.From, subject)
	if len(to) > 0 {
		for _, address := range to {
			address = strings.TrimSpace(address)
			log.Printf("To: %s", address)
			err = c.Rcpt(address)
			if err != nil {
				// Log but continue in case other recipients work out.
				log.Printf("Error: Could not add mail recipient: %q", err.Error())
			}
			message = fmt.Sprintf("%s\nTo: %s", message, address)
		}
	} else {
		log.Print("Potential error: Mail server configured but no recipients specified.")
		log.Printf("Notifying %q", DEFAULTTO)
		err = c.Rcpt(DEFAULTTO)
		if err != nil {
			log.Printf("Error: Could not add mail recipient: %q", err.Error())
			return err
		}
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
// Returns the Index of the new issue created.
func createIssue(job *RegistrationJob, content string, conf *Configuration) (int64, error) {
	repopath := job.Metadata.SourceRepository
	doi := job.Metadata.Identifier.ID
	xmlrepo := job.Config.XMLRepo
	log.Printf("Opening issue on %s", xmlrepo)
	title := fmt.Sprintf("New publication request: %s (%s)", repopath, doi)
	client := job.Config.GIN.Session

	if xmlrepo == "" {
		log.Printf("Issue content body: %s", content)
		return 0, nil
	}

	var resp *http.Response
	var posterr error
	var existingIssue int64
	if issueID, err := getIssueID(client, xmlrepo, title); err == nil {
		if issueID > 0 {
			// Issue exists: Add comment
			path := fmt.Sprintf("api/v1/repos/%s/issues/%d/comments", xmlrepo, issueID)
			data := gogs.CreateIssueCommentOption{Body: content}
			resp, posterr = client.Post(path, data)
			existingIssue = issueID
		} else {
			// Create new issue
			path := fmt.Sprintf("api/v1/repos/%s/issues", xmlrepo)
			data := gogs.CreateIssueOption{
				Title: title,
				Body:  content,
			}
			resp, posterr = client.Post(path, data)
		}
	}
	if posterr != nil {
		log.Printf("Failed to create issue or comment on XML repo: %s", posterr.Error())
		return -1, posterr
	} else if resp.StatusCode != http.StatusCreated {
		var errmsg string
		msg, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errmsg = fmt.Sprintf("Failed to open issue on XML repo: [%d] failed to read response body: %s", resp.StatusCode, err.Error())
		} else {
			errmsg = fmt.Sprintf("Failed to create issue or comment on XML repo: [%d] %s", resp.StatusCode, msg)
		}
		log.Print(errmsg)
		return -1, fmt.Errorf(errmsg)
	}
	if existingIssue > 0 {
		return existingIssue, nil
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Issue creation succeeded, but failed to read response body: %s", err.Error())
	}

	newIssue := new(gogs.Issue)
	err = json.Unmarshal(respBody, newIssue)
	if err != nil {
		log.Printf("Issue creation succeeded, but failed to unmarshal response: %s", err.Error())
		// ignoring error since creation succeeded
		return -1, nil
	}
	return newIssue.Index, nil
}

// getIssueID returns the ID for an issue on a given repo that matches the
// given title. It returns 0 if no issue matching the title is found.
func getIssueID(client *ginclient.Client, repo, title string) (int64, error) {
	path := fmt.Sprintf("api/v1/repos/%s/issues", repo)
	resp, err := client.Get(path)
	if err != nil {
		// log the error and return with -1 and a new issue will be created
		log.Printf("Failed to get issues for repository %s: %s", repo, err.Error())
		return -1, err
	} else if resp.StatusCode != http.StatusOK {
		// log the error and return with -1 and a new issue will be created
		if msg, err := ioutil.ReadAll(resp.Body); err == nil {
			log.Printf("Failed to get issues for repository %s: [%d] %s", repo, resp.StatusCode, msg)
		} else {
			log.Printf("Failed to get issues for repository %s: [%d] failed to read response body: %s", repo, resp.StatusCode, err.Error())
		}
		return -1, err
	}

	var issues []gogs.Issue
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to get issues for repository %s: %s", repo, err.Error())
		return -1, err
	}

	if err := json.Unmarshal(content, &issues); err != nil {
		log.Printf("Failed to get issues for repository %s: failed to unmarshal response: %s", repo, err.Error())
		return -1, err
	}

	for _, issue := range issues {
		if strings.EqualFold(issue.Title, title) {
			log.Printf("Found issue matching %s: %d", title, issue.Index)
			return issue.ID, nil
		}
	}

	return 0, nil
}
