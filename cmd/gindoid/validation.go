package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/G-Node/libgin/libgin"
)

// allowedValues for various keys of the datacite.yml file.
var allowedValues = map[string][]string{
	"reftype":      {"IsSupplementTo", "IsDescribedBy", "IsReferencedBy", "IsVariantFormOf"},
	"resourcetype": {"Dataset", "Software", "DataPaper", "Image", "Text"},
}

// collectWarnings checks for non-critical missing information or issues that
// may need admin attention. These should be sent with the followup
// notification email.
func collectWarnings(job *RegistrationJob) (warnings []string) {
	// Check if any funder IDs are missing
	if job.Metadata.FundingReferences != nil {
		for _, funder := range *job.Metadata.FundingReferences {
			if funder.Identifier == nil || funder.Identifier.ID == "" {
				warnings = append(warnings, fmt.Sprintf("Couldn't find funder ID for funder %q", funder.Funder))
			}
		}
	}

	// Check if a reference from the YAML file uses the old "Name" field instead of "Citation"
	// This shouldn't be an issue, but it can cause formatting issues
	for idx, ref := range job.Metadata.YAMLData.References {
		if ref.Name != "" {
			warnings = append(warnings, fmt.Sprintf("Reference %d uses old 'Name' field instead of 'Citation'", idx))
		}
	}

	// The 80 character limit is arbitrary, but if the abstract is very short, it's worth a check
	if absLen := len(job.Metadata.YAMLData.Description); absLen < 80 {
		warnings = append(warnings, fmt.Sprintf("Abstract may be too short: %d characters", absLen))
	}

	// NOTE: This is a workaround for the current inability to check a
	// potential DOI fork for previous releases.  If the repository has a DOI
	// fork, a notice is added to the admin email to check for previous
	// releases manually.
	if forks, err := getRepoForks(job.Config.GIN.Session, job.Metadata.SourceRepository); err == nil {
		for _, fork := range forks {
			if strings.ToLower(fork.Owner.UserName) == job.Config.GIN.Session.Username {
				warnings = append(warnings, "Repository is already forked by DOI service user: Manual check for releases is required")
				break
			}
		}
	}

	return
}

// checkMissingValues returns a list of messages for missing or invalid values.
// If all values are valid, the returned slice is empty.
func checkMissingValues(info *libgin.RepositoryYAML) []string {
	missing := make([]string, 0, 6)
	if info.Title == "" {
		missing = append(missing, msgNoTitle)
	}
	if len(info.Authors) == 0 {
		missing = append(missing, msgNoAuthors)
	} else {
		for _, auth := range info.Authors {
			if auth.LastName == "" || auth.FirstName == "" {
				missing = append(missing, msgInvalidAuthors)
			}
		}
	}
	if info.Description == "" {
		missing = append(missing, msgNoDescription)
	}
	if info.License == nil || info.License.Name == "" || info.License.URL == "" {
		missing = append(missing, msgNoLicense)
	}
	if info.References != nil {
		for _, ref := range info.References {
			if (ref.Citation == "" && ref.Name == "") || ref.RefType == "" {
				missing = append(missing, msgInvalidReference)
			}
		}
	}
	return missing
}

// checkLicenseMatch returns true if the license text found in the file at the
// URL matches the provided license text. If the file at the URL cannot be
// read, it defaults to true.
func checkLicenseMatch(expectedTextURL string, licenseText string) bool {
	expectedLicenseText, err := readFileAtURL(expectedTextURL)
	if err != nil {
		// License isn't known or there was a problem reading the file in the
		// repository.
		// Return positive response since we can't validate automatically.
		log.Printf("Can't validate License text. Unknown license name in datacite.yml: %q", expectedTextURL)
		return true
	}

	return string(expectedLicenseText) == licenseText
}

func contains(list []string, value string) bool {
	for _, valid := range list {
		if strings.ToLower(valid) == strings.ToLower(value) {
			return true
		}
	}
	return false
}

// validateDataCiteValues checks if the datacite keys that have limited value
// options have a valid value.  Returns a slice of error messages to display to
// the user.  The slice is empty if all values are valid.
func validateDataCiteValues(info *libgin.RepositoryYAML) []string {
	invalid := make([]string, 0)

	if !contains(allowedValues["resourcetype"], info.ResourceType) {
		msg := fmt.Sprintf("<strong>ResourceType</strong> must be one of the following: %s", strings.Join(allowedValues["resourcetype"], ", "))
		invalid = append(invalid, msg)
	}

	for _, ref := range info.References {
		if !contains(allowedValues["reftype"], ref.RefType) {
			msg := fmt.Sprintf("Reference type (<strong>RefType</strong>) must be one of the following: %s", strings.Join(allowedValues["reftype"], ", "))
			invalid = append(invalid, msg)
		}
	}

	return invalid
}
