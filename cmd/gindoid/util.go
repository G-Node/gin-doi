package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/G-Node/libgin/libgin"
)

func readBody(r *http.Request) (*string, error) {
	body, err := ioutil.ReadAll(r.Body)
	x := string(body)
	return &x, err
}

// decrypt from base64 to decrypted string
func decrypt(key []byte, cryptoText string) (string, error) {
	// TODO: Move to libgin
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		return "", err
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return fmt.Sprintf("%s", ciphertext), nil
}

// isRegisteredDOI returns True if a given DOI is registered publicly.
// It simply checks if https://doi.org/<doi> returns a status code other than NotFound.
func isRegisteredDOI(doi string) bool {
	url := fmt.Sprintf("https://doi.org/%s", doi)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Could not query for doi: %s at %s", doi, url)
		return false
	}
	if resp.StatusCode != http.StatusNotFound {
		return true
	}
	return false
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
	names := make([]string, len(authors))
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
			url = author.Identifier.SchemeURI
			id = author.Identifier.ID
		}
		// TODO: Fix URLs
		names[idx] = fmt.Sprintf("<span itemprop=\"author\" itemscope itemtype=\"http://schema.org/Person\"><a href=%q itemprop=\"url\"><span itemprop=\"name\">%s</span></a><meta itemprop=\"affiliation\" content=%q /><meta itemprop=\"identifier\" content=%q>%s</span>", url, author.Name, author.Affiliation, id, affiliationSup)
	}

	authorLine := fmt.Sprintf("<span class=\"doi author\" >\n%s\n</span>", strings.Join(names, ",\n"))
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
