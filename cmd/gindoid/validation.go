package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
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

	// Check authors
	warnings = authorWarnings(job.Metadata.YAMLData, warnings)

	// The 80 character limit is arbitrary, but if the abstract is very short, it's worth a check
	if absLen := len(job.Metadata.YAMLData.Description); absLen < 80 {
		warnings = append(warnings, fmt.Sprintf("Abstract may be too short: %d characters", absLen))
	}

	// Check licenses
	repoLicURL := repoFileURL(job.Config, job.Metadata.SourceRepository, "LICENSE")
	warnings = licenseWarnings(job.Metadata.YAMLData, repoLicURL, warnings)

	// Check if any funder IDs are missing
	if job.Metadata.FundingReferences != nil {
		for _, funder := range *job.Metadata.FundingReferences {
			if funder.Identifier == nil || funder.Identifier.ID == "" {
				warnings = append(warnings, fmt.Sprintf("Couldn't find funder ID for funder %q", funder.Funder))
			}
		}
	}

	// Check references
	warnings = referenceWarnings(job.Metadata.YAMLData, warnings)

	return
}

// authorWarnings checks datacite authors for validity and returns
// corresponding warnings if required.
func authorWarnings(yada *libgin.RepositoryYAML, warnings []string) []string {
	var orcidRE = regexp.MustCompile(`([[:digit:]]{4}-){3}[[:digit:]]{3}[[:digit:]X]`)
	var dupID = make(map[string]string)

	for idx, auth := range yada.Authors {
		if auth.ID == "" {
			continue
		}
		lowerID := strings.ToLower(auth.ID)

		// Warn when not able to identify ID type
		if !strings.HasPrefix(lowerID, "orcid") && !strings.HasPrefix(lowerID, "researcherid") {
			if orcid := orcidRE.Find([]byte(auth.ID)); orcid != nil {
				warnings = append(warnings, fmt.Sprintf("Author %d (%s) has ORCID-like unspecified ID: %s", idx, auth.LastName, auth.ID))
			} else {
				warnings = append(warnings, fmt.Sprintf("Author %d (%s) has unknown ID: %s", idx, auth.LastName, auth.ID))
			}
		}

		// Warn on known ID type but missing value
		idpref := map[string]bool{"orcid:": true, "researcherid:": true}
		if _, found := idpref[strings.TrimSpace(lowerID)]; found {
			warnings = append(warnings, fmt.Sprintf("Author %d (%s) has empty ID value: %s", idx, auth.LastName, auth.ID))
		}

		// Warn on dupliate ID entries
		if authName, isduplicate := dupID[lowerID]; isduplicate {
			curr := fmt.Sprintf("%d (%s)", idx, auth.LastName)
			warnings = append(warnings, fmt.Sprintf("Authors %s and %s have the same ID: %s", authName, curr, auth.ID))
		} else {
			dupID[lowerID] = fmt.Sprintf("%d (%s)", idx, auth.LastName)
		}
	}

	return warnings
}

// referenceWarnings checks datacite references for validity and
// returns corresponding warnings if required.
func referenceWarnings(yada *libgin.RepositoryYAML, warnings []string) []string {
	for idx, ref := range yada.References {
		// Check if a reference from the YAML file uses the old "Name" field instead of "Citation"
		// This shouldn't be an issue, but it can cause formatting issues
		if ref.Name != "" {
			warnings = append(warnings, fmt.Sprintf("Reference %d uses old 'Name' field instead of 'Citation'", idx))
		}

		// Warn if reftypes are different from "IsSupplementTo"
		if strings.ToLower(ref.RefType) != "issupplementto" {
			warnings = append(warnings, fmt.Sprintf("Reference %d uses refType '%s'", idx, ref.RefType))
		}

		// Warn if a reference does not provide a relatedIdentifier
		var relIDType string
		refIDParts := strings.SplitN(ref.ID, ":", 2)
		if len(refIDParts) == 2 {
			relIDType = strings.TrimSpace(refIDParts[0])
		}
		if relIDType == "" {
			warnings = append(warnings, fmt.Sprintf("Reference %d has no related ID type: '%s'; excluded from XML file", idx, ref.ID))
		}
	}
	return warnings
}

// DOILicense holds Name (official license title), URL (license online reference)
// and Alias names for a license used for a DOI registration.
type DOILicense struct {
	URL   string
	Name  string
	Alias []string
}

// licenseWarnings checks license URL, name and license content header
// for consistency and against common licenses.
func licenseWarnings(yada *libgin.RepositoryYAML, repoLicenseURL string, warnings []string) []string {
	// check datacite license URL, name and license file title to spot mismatches
	commonLicenses := ReadCommonLicenses()

	// check if the datacite license can be matched to a common license via URL
	licenseURL, ok := licFromURL(commonLicenses, yada.License.URL)
	if !ok {
		warnings = append(warnings, fmt.Sprintf("License URL (datacite) not found: '%s'", yada.License.URL))
	}

	// check if the license can be matched to a common license via datacite license name
	licenseName, ok := licFromName(commonLicenses, yada.License.Name)
	if !ok {
		warnings = append(warnings, fmt.Sprintf("License name (datacite) not found: '%s'", yada.License.Name))
	}

	// check if the license can be matched to a common license via the header line of the license file
	var licenseHeader DOILicense
	content, err := readFileAtURL(repoLicenseURL)
	if err != nil {
		warnings = append(warnings, "Could not access license file")
	} else {
		headstr := string(content)
		fileHeader := strings.Split(strings.Replace(headstr, "\r\n", "\n", -1), "\n")
		var ok bool // false if fileHeader 0 or licFromName returns !ok
		if len(fileHeader) > 0 {
			licenseHeader, ok = licFromName(commonLicenses, fileHeader[0])
		}
		if !ok {
			// Limit license file content in warning message
			if len(headstr) > 20 {
				headstr = fmt.Sprintf("%s...", headstr[0:20])
			}
			warnings = append(warnings, fmt.Sprintf("License file content header not found: '%s'", headstr))
		}
	}

	// check license URL against license name
	if licenseURL.Name != licenseName.Name {
		warnings = append(warnings, fmt.Sprintf("License URL/Name mismatch: '%s'/'%s'", licenseURL.Name, licenseName.Name))
	}

	// check license name against license file header
	if licenseName.Name != licenseHeader.Name {
		warnings = append(warnings, fmt.Sprintf("License name/file header mismatch: '%s'/'%s'", licenseName.Name, licenseHeader.Name))
	}

	return warnings
}

// licFromURL identifies a common license from a []DOILicense via a specified license URL.
// Returns either the found or an empty DOILicense and a corresponding boolean 'ok' flag.
func licFromURL(commonLicenses []DOILicense, licenseURL string) (DOILicense, bool) {
	url := cleancompstr(licenseURL)
	for _, lic := range commonLicenses {
		// provided licenses URLs can be more verbose than the default license URL
		if strings.Contains(url, strings.ToLower(lic.URL)) {
			return lic, true
		}
	}

	var emptyLicense DOILicense
	return emptyLicense, false
}

// licFromName identifies a common license from a []DOILicense via a specific license title.
// Returns either the found or an empty DOILicense and a corresponding boolean 'ok' flag.
func licFromName(commonLicenses []DOILicense, licenseName string) (DOILicense, bool) {
	licname := cleancompstr(licenseName)
	for _, lic := range commonLicenses {
		for _, alias := range lic.Alias {
			if licname == strings.ToLower(alias) {
				return lic, true
			}
		}
	}

	var emptyLicense DOILicense
	return emptyLicense, false
}

// ReadCommonLicenses returns an array of common DOI licenses.
// The common DOI licenses are read from a "doi-licenses.json"
// file found besides the DOI environment variables file. This
// enables an update to common DOI licenses without restarting
// the server.
// If this file is not available, a fallback license file
// is read from a resources folder. If none of the license files
// can be read, an empty []DOILicense is returned.
func ReadCommonLicenses() []DOILicense {
	// try to load custom license file from the env var directory
	filepath := filepath.Join(libgin.ReadConf("configdir"), "doi-licenses.json")
	licenses, err := licenseFromFile(filepath)
	if err == nil {
		log.Println("Using custom licenses")
		return licenses
	}

	// if a custom license is not available, fetch default licenses
	var defaultLicenses []DOILicense
	if err = json.Unmarshal([]byte(defaultLicensesJSON), &defaultLicenses); err == nil {
		log.Println("Using default licenses")
		return defaultLicenses
	}

	// everything failed, return empty licenses struct
	log.Println("Could not load licenses")
	var emptyLicenses []DOILicense
	return emptyLicenses
}

// licenseFromFile opens a file from a provided filepath and
// json unmarshals the file contents into a []DOILicense.
// Returns an error or a valid []DOILicense.
func licenseFromFile(filepath string) ([]DOILicense, error) {
	fp, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	jdata, err := ioutil.ReadAll(fp)
	if err != nil {
		return nil, err
	}

	var licenses []DOILicense
	if err = json.Unmarshal(jdata, &licenses); err != nil {
		return nil, err
	}

	return licenses, nil
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

// cleancompstr cleans up an input string.
// Surrounding whitespaces are removed and
// converted to lower case.
func cleancompstr(cleanup string) string {
	cleanup = strings.TrimSpace(cleanup)
	cleanup = strings.ToLower(cleanup)
	return cleanup
}
