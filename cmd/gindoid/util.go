package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	gdtmpl "github.com/G-Node/gin-doi/templates"
	"github.com/G-Node/libgin/libgin"
)

// ALNUM provides characters for the randAlnum function.
const ALNUM = "1234567890abcdefghijklmnopqrstuvwxyz"

// randAlnum returns a random alphanumeric (lowercase, latin) string of length 'n'.
func randAlnum(n int) string {
	N := len(ALNUM)

	chrs := make([]byte, n)
	rand.Seed(time.Now().UnixNano())
	for idx := range chrs {
		chrs[idx] = ALNUM[rand.Intn(N)]
	}

	return string(chrs)
}

// isURL returns true if a URL scheme part can be identfied
// within a passed string. Returns false in any other case.
func isURL(str string) bool {
	if purl, err := url.Parse(str); err == nil {
		return purl.Scheme != ""
	}
	return false
}

// deduplicateValues checks a string slice for duplicate
// entries and returns a reduced string slice without any
// duplicates.
func deduplicateValues(dupvals []string) []string {
	strmap := make(map[string]bool)
	vals := []string{}
	for _, val := range dupvals {
		if _, exists := strmap[val]; !exists {
			strmap[val] = true
			vals = append(vals, val)
		}
	}
	return vals
}

// readFileAtPath returns the content of a file at a given path.
func readFileAtPath(path string) ([]byte, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer fp.Close()

	stat, err := fp.Stat()
	if err != nil {
		return nil, err
	}
	contents := make([]byte, stat.Size())
	_, err = fp.Read(contents)
	return contents, err
}

// readFileAtURL returns the contents of a file at a given URL.
func readFileAtURL(url string) ([]byte, error) {
	client := &http.Client{}
	log.Printf("Fetching file at %q", url)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Request failed: %s", err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request returned non-OK status: %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Could not read file contents: %s", err.Error())
		return nil, err
	}
	return body, nil
}

// EscXML runs a string through xml.EscapeText.
// This is a utility function for the doi.xml template.
func EscXML(txt string) string {
	buf := new(bytes.Buffer)
	if err := xml.EscapeText(buf, []byte(txt)); err != nil {
		log.Printf("Could not escape: %q :: %s", txt, err.Error())
		return ""
	}
	return buf.String()
}

// ReferenceDescription creates a string representation of a reference for use in the XML description tag.
// This is a utility function for the doi.xml template.
func ReferenceDescription(ref libgin.Reference) string {
	var namecitation string
	if ref.Name != "" && ref.Citation != "" {
		namecitation = ref.Name + " " + ref.Citation
	} else {
		namecitation = ref.Name + ref.Citation
	}

	if !strings.HasSuffix(namecitation, ".") {
		namecitation += "."
	}
	refDesc := fmt.Sprintf("%s: %s (%s)", ref.RefType, namecitation, ref.ID)
	return EscXML(refDesc)
}

// ReferenceSource splits the source type from a reference string of the form <source>:<ID>
// This is a utility function for the doi.xml template.
func ReferenceSource(ref libgin.Reference) string {
	idparts := strings.SplitN(ref.ID, ":", 2)
	if len(idparts) != 2 {
		// Malformed ID (no colon)
		// No source type
		return ""
	}
	return EscXML(idparts[0])
}

// ReferenceID splits the ID from a reference string of the form <source>:<ID>
// This is a utility function for the doi.xml template.
func ReferenceID(ref libgin.Reference) string {
	idparts := strings.SplitN(ref.ID, ":", 2)
	if len(idparts) != 2 {
		// Malformed ID (no colon)
		// No source type
		return EscXML(idparts[0])
	}
	return EscXML(idparts[1])
}

// GetGINURL returns the full URL to the configured GIN server. If it's
// configured with a non-standard port, the port number is included.
func GetGINURL(conf *Configuration) string {
	address := conf.GIN.Session.WebAddress()
	// get scheme
	schemeSepIdx := strings.Index(address, "://")
	if schemeSepIdx == -1 {
		// no scheme; return as is
		return address
	}
	// get port
	portSepIdx := strings.LastIndex(address, ":")
	if portSepIdx == -1 {
		// no port; return as is
		return address
	}
	scheme := address[:schemeSepIdx]
	port := address[portSepIdx:len(address)]
	if (scheme == "http" && port == ":80") ||
		(scheme == "https" && port == ":443") {
		// port is standard for scheme: slice it off
		address = address[0:portSepIdx]
	}
	return address
}

var templateMap = map[string]string{
	"Nav":                gdtmpl.Nav,
	"Footer":             gdtmpl.Footer,
	"RequestFailurePage": gdtmpl.RequestFailurePage,
	"RequestPage":        gdtmpl.RequestPage,
	"RequestResult":      gdtmpl.RequestResult,
	"DOIInfo":            gdtmpl.DOIInfo,
	"LandingPage":        gdtmpl.LandingPage,
	"KeywordIndex":       gdtmpl.KeywordIndex,
	"Keyword":            gdtmpl.Keyword,
}

// prepareTemplates initialises and parses a sequence of templates in the order
// they appear in the arguments.  It always adds the Nav template first and
// includes the common template functions in tmplfuncs.
func prepareTemplates(templateNames ...string) (*template.Template, error) {
	tmpl, err := template.New("Nav").Funcs(tmplfuncs).Parse(templateMap["Nav"])
	if err != nil {
		log.Printf("Could not parse the \"Nav\" template: %s", err.Error())
		return nil, err
	}
	tmpl, err = tmpl.New("Footer").Parse(templateMap["Footer"])
	if err != nil {
		log.Printf("Could not parse the \"Footer\" template: %s", err.Error())
		return nil, err
	}
	for _, tName := range templateNames {
		tContent, ok := templateMap[tName]
		if !ok {
			return nil, fmt.Errorf("unknown template with name %q", tName)
		}
		tmpl, err = tmpl.New(tName).Parse(tContent)
		if err != nil {
			log.Printf("Could not parse the %q template: %s", tName, err.Error())
			return nil, err
		}
	}
	return tmpl, nil
}

// Global function map for the templates that render the DOI information
// (request page and landing page).
var tmplfuncs = template.FuncMap{
	"Upper":            strings.ToUpper,
	"FunderName":       FunderName,
	"AwardNumber":      AwardNumber,
	"AuthorBlock":      AuthorBlock,
	"JoinComma":        JoinComma,
	"Replace":          strings.ReplaceAll,
	"FormatReferences": FormatReferences,
	"FormatCitation":   FormatCitation,
	"FormatIssuedDate": FormatIssuedDate,
	"KeywordPath":      KeywordPath,
	"FormatAuthorList": FormatAuthorList,
	"NewVersionNotice": NewVersionNotice,
	"OldVersionLink":   OldVersionLink,
	"GINServerURL":     GINServerURL,
}

// FunderName splits the funder name from a funding string of the form <FunderName>; <AwardNumber>.
// This is a utility function for the doi.xml template. Split character is semi-colon,
// but for backwards compatibility reasons, comma is supported as a fallback split character
// if no semi-colon is provided.
func FunderName(fundref string) string {
	splitchar := ";"
	if !strings.Contains(fundref, splitchar) {
		splitchar = ","
	}

	fuparts := strings.SplitN(fundref, splitchar, 2)
	if len(fuparts) != 2 {
		// No splitchar, return as is
		return EscXML(fundref)
	}
	return EscXML(strings.TrimSpace(fuparts[0]))
}

// AwardNumber splits the award number from a funding string of the form <FunderName>; <AwardNumber>.
// This is a utility function for the doi.xml template. Split character is semi-colon,
// but for backwards compatibility reasons, comma is supported as a fallback split character
// if no semi-colon is provided.
func AwardNumber(fundref string) string {
	splitchar := ";"
	if !strings.Contains(fundref, splitchar) {
		splitchar = ","
	}

	fuparts := strings.SplitN(fundref, splitchar, 2)
	if len(fuparts) != 2 {
		// No splitchar, return empty
		return ""
	}
	return EscXML(strings.TrimSpace(fuparts[1]))
}

// AuthorBlock builds the author section for the landing page template.
// It includes a list of authors, their affiliations, and superscripts to associate authors with affiliations.
// This is a utility function for the landing page HTML template.
func AuthorBlock(authors []libgin.Creator) template.HTML {
	authorMap := make(map[*libgin.Creator]int) // maps Author -> Affiliation Number
	affiliationMap := make(map[string]int)     // maps Affiliation Name -> Affiliation Number
	affilNumberMap := make(map[int]string)     // maps Affiliation Number -> Affiliation Name (inverse of above)
	for _, author := range authors {
		if _, ok := affiliationMap[author.Affiliation]; !ok {
			// new affiliation; give it a new number
			number := 0
			// NOTE: adding the empty affiliation helps us figure out if a
			// single unique affiliation should be numbered, since we should
			// differentiate between authors that share the affiliation and the
			// ones that have none.
			if author.Affiliation != "" {
				number = len(affiliationMap) + 1
			} // otherwise it gets the "special" value 0
			affiliationMap[author.Affiliation] = number
			affilNumberMap[number] = author.Affiliation
		}
		authorMap[&author] = affiliationMap[author.Affiliation]
	}

	nameElements := make([]string, len(authors))
	// Format authors
	for idx, author := range authors {
		var url, id, affiliationSup string
		if author.Identifier != nil {
			id = author.Identifier.ID
			if author.Identifier.SchemeURI != "" {
				url = author.Identifier.SchemeURI + id
			}
		}

		// Author names are LastName, FirstName
		namesplit := strings.SplitN(author.Name, ",", 2)
		name := author.Name
		// If there's no comma, just display as is
		if len(namesplit) == 2 {
			name = fmt.Sprintf("%s %s", strings.TrimSpace(namesplit[1]), strings.TrimSpace(namesplit[0]))
		}

		// Add superscript to name if it has an affiliation and there are more than one (including empty)
		if author.Affiliation != "" && len(affiliationMap) > 1 {
			affiliationSup = fmt.Sprintf("<sup>%d</sup>", affiliationMap[author.Affiliation])
		}

		nameElements[idx] = fmt.Sprintf("<span itemprop=\"author\" itemscope itemtype=\"http://schema.org/Person\"><a href=%q itemprop=\"url\"><span itemprop=\"name\">%s</span></a><meta itemprop=\"affiliation\" content=%q /><meta itemprop=\"identifier\" content=%q>%s</span>", url, name, author.Affiliation, id, affiliationSup)
	}

	// Format affiliations in number order (excluding empty)
	affiliationLines := "<ol class=\"doi itemlist\">\n"
	for idx := 1; ; idx++ {
		affiliation, ok := affilNumberMap[idx]
		if !ok {
			break
		}
		var supstr string
		if len(affiliationMap) > 1 {
			supstr = fmt.Sprintf("<sup>%d</sup>", idx)
		}
		affiliationLines = fmt.Sprintf("%s\t<li>%s%s</li>\n", affiliationLines, supstr, affiliation)
	}
	affiliationLines = fmt.Sprintf("%s</ol>", affiliationLines)

	authorLines := fmt.Sprintf("<span class=\"doi author\" >\n%s\n</span>", strings.Join(nameElements, ",\n"))
	return template.HTML(authorLines + "\n" + affiliationLines)
}

// JoinComma joins a slice of strings into a single string separated by commas
// (and space).  Useful for generating comma-separated lists of entries for
// templates.
func JoinComma(lst []string) string {
	return strings.Join(lst, ", ")
}

// FormatReferences returns the references cited by a dataset.  If the references
// are already populated in the YAMLData field they are returned as is.  If
// they are not, they are reconstructed to the YAML format from the DataCite
// metadata.  The latter can occur when loading a previously generated DataCite
// XML file instead of reading the original YAML from the repository.  If no
// references are found in either location, an empty slice is returned.
func FormatReferences(md *libgin.RepositoryMetadata) []libgin.Reference {
	if md.YAMLData != nil && len(md.YAMLData.References) != 0 {
		return md.YAMLData.References
	}

	// No references in YAML data; reconstruct from DataCite metadata if any
	// are found.

	// collect reference descriptions (descriptionType="Other")
	referenceDescriptions := make([]string, 0)
	for _, desc := range md.Descriptions {
		if desc.Type == "Other" {
			referenceDescriptions = append(referenceDescriptions, desc.Content)
		}
	}

	findDescriptionIdx := func(id string) int {
		for idx, desc := range referenceDescriptions {
			if strings.Contains(desc, id) {
				return idx
			}
		}
		return -1
	}

	splitDescriptionType := func(desc string) (string, string) {
		descParts := strings.SplitN(desc, ":", 2)
		if len(descParts) != 2 {
			return "", desc
		}

		return strings.TrimSpace(descParts[0]), strings.TrimSpace(descParts[1])
	}

	refs := make([]libgin.Reference, 0)
	// map IDs to new references for easier construction from the two sources
	// but also use the slice to maintain order
	for _, relid := range md.RelatedIdentifiers {
		if relid.RelationType == "IsVariantFormOf" || relid.RelationType == "IsIdenticalTo" ||
			relid.RelationType == "IsNewVersionOf" || relid.RelationType == "IsOldVersionOf" {
			// IsVariantFormOf is used for the URLs.
			// IsIdenticalTo is used for the old DOI URLs.
			// Is{New,Old}Version of is used for {new,old} versions of the same dataset.
			// Here we assume that any other type is a citation
			continue
		}
		ref := &libgin.Reference{
			ID:      fmt.Sprintf("%s:%s", relid.Type, relid.Identifier),
			RefType: relid.RelationType,
		}
		if idx := findDescriptionIdx(relid.Identifier); idx >= 0 {
			citation := referenceDescriptions[idx]
			referenceDescriptions = append(referenceDescriptions[:idx], referenceDescriptions[idx+1:]...) // remove found element
			_, citation = splitDescriptionType(citation)
			// filter out the DOI URL from the citation
			urlstr := fmt.Sprintf("(%s)", ref.GetURL())
			citation = strings.Replace(citation, urlstr, "", -1)
			ref.Citation = citation
		}
		refs = append(refs, *ref)
	}

	// Add the rest of the descriptions that didn't have an ID to match (if any)
	for _, refDesc := range referenceDescriptions {
		refType, citation := splitDescriptionType(refDesc)
		ref := libgin.Reference{
			ID:       "",
			RefType:  refType,
			Citation: citation,
		}
		refs = append(refs, ref)
	}
	if len(refs) == 0 {
		return nil
	}
	return refs
}

// FormatCitation returns the formatted citation string for a given dataset.
func FormatCitation(md *libgin.RepositoryMetadata) string {
	authors := make([]string, len(md.Creators))
	for idx, author := range md.Creators {
		namesplit := strings.SplitN(author.Name, ",", 2) // Author names are LastName, FirstName
		if len(namesplit) != 2 {
			// No comma: Bad input, mononym, or empty field.
			// Trim, add continue.
			authors[idx] = strings.TrimSpace(author.Name)
			continue
		}
		// render as LastName Initials, ...
		firstnames := strings.Fields(namesplit[1])
		var initials string
		for _, name := range firstnames {
			initials += string(name[0])
		}
		authors[idx] = fmt.Sprintf("%s %s", strings.TrimSpace(namesplit[0]), initials)
	}
	return fmt.Sprintf("%s (%d) %s. G-Node. https://doi.org/%s", strings.Join(authors, ", "), md.Year, md.Titles[0], md.Identifier.ID)
}

// FormatIssuedDate returns the issued date of the dataset in the format DD Mon.
// YYYY for adding to the preparation and landing pages.
func FormatIssuedDate(md *libgin.RepositoryMetadata) string {
	var datestr string
	for _, mddate := range md.Dates {
		// There should be only one, but we might add some other types of date
		// at some point, so best be safe.
		if mddate.Type == "Issued" {
			datestr = mddate.Value
			break
		}
	}

	date, err := time.Parse("2006-01-02", datestr)
	if err != nil {
		// This will also occur if the date isn't found in 'md' and the string
		// remains empty
		log.Printf("Failed to parse issued date: %s", datestr)
		return ""
	}
	return date.Format("02 Jan. 2006")
}

// KeywordPath returns a keyword sanitised for use in a URL path:
// Lowercase + replace / with _.
func KeywordPath(kw string) string {
	kw = strings.ToLower(kw)
	kw = strings.ReplaceAll(kw, "/", "_")
	return kw
}

// FormatAuthorList returns a comma-separated list of the author names for a
// dataset.
func FormatAuthorList(md *libgin.RepositoryMetadata) string {
	names := make([]string, len(md.Creators))
	for idx, author := range md.Creators {
		names[idx] = author.Name
	}
	authors := strings.Join(names, ", ")
	return authors
}

// NewVersionNotice returns an HTML template containing links to a newer version
// of a given dataset if it exists.
func NewVersionNotice(md *libgin.RepositoryMetadata) template.HTML {
	for _, relid := range md.RelatedIdentifiers {
		if relid.RelationType == "IsOldVersionOf" {
			noticeContainer := `
<div class="ui warning message">
	<div class="header">New dataset version</div>
	<p>%s<p>
</div>`
			url := fmt.Sprintf("https://doi.org/%s", relid.Identifier)
			text := fmt.Sprintf("A newer version of this dataset is available at <a href=%s>%s</a>", url, url)
			return template.HTML(fmt.Sprintf(noticeContainer, text))
		}
	}
	return ""
}

// OldVersionLink returns an HTML template containing links to a previous version
// of a given dataset if it exists.
func OldVersionLink(md *libgin.RepositoryMetadata) template.HTML {
	for _, relid := range md.RelatedIdentifiers {
		if relid.RelationType == "IsNewVersionOf" {
			url := fmt.Sprintf("https://doi.org/%s", relid.Identifier)
			header := "<h3>Versions</h3>"
			text := fmt.Sprintf("An older version of this dataset is available at <a href=%s>%s</a>", url, url)
			return template.HTML(header + "\n" + text)
		}
	}
	return ""
}

// GINServerURL is the default template function returning
// the main GIN server URL.  This function can be overriden
// before calling HTML template execution to provide a different
// GIN server instance URL.
func GINServerURL() string {
	return "https://gin.g-node.org"
}
