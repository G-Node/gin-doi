package main

const doiXML = `<?xml version="1.0" encoding="UTF-8"?>
<resource xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://datacite.org/schema/kernel-4" xsi:schemaLocation="http://datacite.org/schema/kernel-4 http://schema.datacite.org/meta/kernel-4/metadata.xsd">
  <identifier identifierType="DOI">{{.DOI}}</identifier>
  <creators>{{range $index, $auth := .Authors}}
    <creator>
      <creatorName>{{EscXML $auth.LastName}}, {{EscXML $auth.FirstName}}</creatorName>
        {{if $auth.GetValidID}}<nameIdentifier schemeURI="{{$auth.GetValidID.URI}}" nameIdentifierScheme="{{$auth.GetValidID.Scheme}}">{{EscXML $auth.GetValidID.ID}}</nameIdentifier>{{end}}
        {{if $auth.Affiliation}}<affiliation>{{EscXML $auth.Affiliation}}</affiliation>{{end}}
    </creator>{{end}}
  </creators>
  <titles>
    <title>{{EscXML .Title}}</title>
  </titles>
  {{if or .Description .References}}<descriptions>
    {{if .Description}}<description descriptionType="Abstract">
      {{EscXML .Description}}
    </description>{{end}}
    {{range $index, $ref := .References}}<description descriptionType="Other">
      {{ReferenceDescription $ref}}
    </description>{{end}}
  </descriptions>{{end}}
  {{if .License}}<rightsList>
    <rights {{if .License.URL}}rightsURI="{{.License.URL}}"{{end}}> {{EscXML .License.Name}}</rights>
  </rightsList>{{end}}
  {{if .Keywords}}<subjects>{{range $index, $kw := .Keywords}}
     <subject>{{EscXML $kw}}</subject>{{end}}
  </subjects>{{end}}
  {{if .References}}<relatedIdentifiers>{{range $index, $ref := .References}}
    <relatedIdentifier relatedIdentifierType="{{ReferenceSource $ref}}" relationType="{{$ref.Reftype}}">{{ReferenceID $ref}}</relatedIdentifier>{{end}}
  </relatedIdentifiers>{{end}}
  {{if .Funding}}<fundingReferences>{{range $index, $fu := .Funding}}
    <fundingReference><funderName>{{FunderName $fu}}</funderName><awardNumber>{{AwardNumber $fu}}</awardNumber></fundingReference>{{end}}
  </fundingReferences>{{end}}
  <contributors>
    <contributor contributorType="HostingInstitution">
      <contributorName>German Neuroinformatics Node</contributorName>
    </contributor>
  </contributors>
  <publisher>G-Node</publisher>
  <publicationYear>{{.Year}}</publicationYear>
  <dates>
      <date dateType="Issued">{{.ISODate}}</date>
  </dates>
  <language>eng</language>
  <resourceType resourceTypeGeneral="{{.GetType}}">{{.GetType}}</resourceType>
  <version>1</version>
</resource>`
