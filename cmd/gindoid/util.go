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
	return fmt.Sprintf("%s: %s (%s)", ref.Reftype, namecitation, ref.ID)
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
	return idparts[0]
}

// ReferenceID splits the ID from a reference string of the form <source>:<ID>
// This is a utility function for the doi.xml template.
func ReferenceID(ref libgin.Reference) string {
	idparts := strings.SplitN(ref.ID, ":", 2)
	if len(idparts) != 2 {
		// Malformed ID (no colon)
		// No source type
		return idparts[0]
	}
	return idparts[1]
}

// FunderName splits the funder name from a funding string of the form <FunderName>, <AwardNumber>.
// This is a utility function for the doi.xml template.
func FunderName(fundref string) string {
	fuparts := strings.SplitN(fundref, ",", 2)
	if len(fuparts) != 2 {
		// No comma, return as is
		return fundref
	}
	return strings.TrimSpace(fuparts[0])
}

// AwardNumber splits the award number from a funding string of the form <FunderName>, <AwardNumber>.
// This is a utility function for the doi.xml template.
func AwardNumber(fundref string) string {
	fuparts := strings.SplitN(fundref, ",", 2)
	if len(fuparts) != 2 {
		// No comma, return empty
		return ""
	}
	return strings.TrimSpace(fuparts[1])
}
