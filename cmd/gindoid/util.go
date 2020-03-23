package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/G-Node/libgin/libgin"
)

// Global function map for the templates that render the DOI information
// (request page and landing page).
var tmplfuncs = template.FuncMap{
	"Upper":         strings.ToUpper,
	"FunderName":    FunderName,
	"AwardNumber":   AwardNumber,
	"AuthorBlock":   AuthorBlock,
	"JoinComma":     JoinComma,
	"Replace":       strings.ReplaceAll,
	"GetReferences": GetReferences,
	"GetCitation":   GetCitation,
	"GetIssuedDate": GetIssuedDate,
}

func readBody(r *http.Request) (*string, error) {
	body, err := ioutil.ReadAll(r.Body)
	x := string(body)
	return &x, err
}

func makeUUID(URI string) string {
	if doi, ok := libgin.UUIDMap[URI]; ok {
		return doi
	}
	currMd5 := md5.Sum([]byte(URI))
	return hex.EncodeToString(currMd5[:])
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

// FunderName splits the funder name from a funding string of the form <FunderName>, <AwardNumber>.
// This is a utility function for the doi.xml template.
func FunderName(fundref string) string {
	fuparts := strings.SplitN(fundref, ",", 2)
	if len(fuparts) != 2 {
		// No comma, return as is
		return EscXML(fundref)
	}
	return EscXML(strings.TrimSpace(fuparts[0]))
}

// AwardNumber splits the award number from a funding string of the form <FunderName>, <AwardNumber>.
// This is a utility function for the doi.xml template.
func AwardNumber(fundref string) string {
	fuparts := strings.SplitN(fundref, ",", 2)
	if len(fuparts) != 2 {
		// No comma, return empty
		return ""
	}
	return EscXML(strings.TrimSpace(fuparts[1]))
}

// AuthorBlock builds the author section for the landing page template.
// It includes a list of authors, their affiliations, and superscripts to associate authors with affiliations.
// This is a utility function for the landing page HTML template.
func AuthorBlock(authors []libgin.Creator) template.HTML {
	nameElements := make([]string, len(authors))
	affiliations := make([]string, 0)
	affiliationMap := make(map[string]int)
	// Collect names and figure out affiliation numbering
	for idx, author := range authors {
		var affiliationSup string // if there's no affiliation, don't add a superscript
		if author.Affiliation != "" {
			if _, ok := affiliationMap[author.Affiliation]; !ok {
				// new affiliation; give it a new number, otherwise the existing one will be used below
				num := len(affiliationMap) + 1
				affiliationMap[author.Affiliation] = num
				affiliations = append(affiliations, fmt.Sprintf("<li><sup>%d</sup>%s</li>", num, author.Affiliation))
			}
			affiliationSup = fmt.Sprintf("<sup>%d</sup>", affiliationMap[author.Affiliation])
		}
		var url, id string
		if author.Identifier != nil {
			id = author.Identifier.ID
			url = author.Identifier.SchemeURI + id
		}
		namesplit := strings.SplitN(author.Name, ",", 2) // Author names are LastName, FirstName
		name := fmt.Sprintf("%s %s", namesplit[1], namesplit[0])
		nameElements[idx] = fmt.Sprintf("<span itemprop=\"author\" itemscope itemtype=\"http://schema.org/Person\"><a href=%q itemprop=\"url\"><span itemprop=\"name\">%s</span></a><meta itemprop=\"affiliation\" content=%q /><meta itemprop=\"identifier\" content=%q>%s</span>", url, name, author.Affiliation, id, affiliationSup)
	}

	authorLine := fmt.Sprintf("<span class=\"doi author\" >\n%s\n</span>", strings.Join(nameElements, ",\n"))
	affiliationLine := fmt.Sprintf("<ol class=\"doi itemlist\">%s</ol>", strings.Join(affiliations, "\n"))
	return template.HTML(authorLine + "\n" + affiliationLine)
}

// JoinComma joins a slice of strings into a single string separated by commas
// (and space).  Useful for generating comma-separated lists of entries for
// templates.
func JoinComma(lst []string) string {
	return strings.Join(lst, ", ")
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

func GetCitation(md *libgin.RepositoryMetadata) string {
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
	return fmt.Sprintf("%s (%d) %s. G-Node. doi:%s", strings.Join(authors, ", "), md.Year, md.Titles[0], md.Identifier.ID)
}

// GetReferences returns the references cited by a dataset.  If the references
// are already populated in the YAMLData field they are returned as is.  If
// they are not, they are reconstructed to the YAML format from the DataCite
// metadata.  The latter can occur when loading a previously generated DataCite
// XML file instead of reading the original YAML from the repository.  If no
// references are found in either location, an empty slice is returned.
func GetReferences(md *libgin.RepositoryMetadata) []libgin.Reference {
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
		if relid.RelationType == "IsVariantFormOf" {
			// IsVariantFormOf is used for the URLs.
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
			// filter out the ID from the citation
			idstr := fmt.Sprintf("(%s)", ref.ID)
			citation = strings.Replace(citation, idstr, "", -1)
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

// GetIssuedDate returns the issued date of the dataset in the format DD Mon.
// YYYY for adding to the preparation and landing pages.
func GetIssuedDate(md *libgin.RepositoryMetadata) string {
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
