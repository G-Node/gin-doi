package main

const requestPageTmpl = `<!DOCTYPE html>
<html lang="en">
	<head data-suburl="">

		<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">

		<meta name="robots" content="noindex,nofollow">


		<meta name="author" content="G-Node">
		<meta name="description" content="Info">
		<meta name="keywords" content="gin, data, sharing, science git">

		<meta property="og:url" content="https://gin.g-node.org/G-Node/Info">
		<meta property="og:type" content="object">
		<meta property="og:title" content="G-Node/Info">
		<meta property="og:description" content="">
		<meta property="og:image" content="https://gin.g-node.org/avatars/18">


		<link rel="shortcut icon" href="/assets/img/favicon.png">
		<link rel="stylesheet" href="/assets/octicons-4.3.0/octicons.min.css">
		<link rel="stylesheet" href="/assets/css/semantic-2.3.1.min.css">
		<link rel="stylesheet" href="/assets/css/gogs.css">
		<link rel="stylesheet" href="/assets/css/custom.css">

		<script src="/assets/js/jquery-1.11.3.min.js"></script>
		<script>
			$(document).ready(function () {
				$ = jQuery.noConflict();
				$.ajaxSetup({cache: false});
				$("#main").on('click', "#doify", doify);
			});

function doify(event) {
	$(event.target).addClass("disabled");
	$.ajax({
		url: "/do/",
		type: "POST",
		contentType: "text/plain",
		data: "{\"repository\":\"{{.Repository}}\",\"username\":\"{{.Username}}\",\"verification\":\"{{.Verification}}\"}",
		dataType: "text",
		success: function (data) {
			$("#info").html($.parseHTML(data));
			$("#info").toggleClass("ui positive message");
			$("#info").toggleClass("ui info icon message");
			$("#infotable").hide();
			$("#warning").hide();
		},
		error: function (data) {
			$("#info").html("An internal error occured while we were processing your request.  The G-Node team has been notified of the problem and will attempt to repair it and process your request.  We may contact you for further information regarding your request.  Feel free to <a href=mailto:gin@g-node.org>contact us</a> if you would like to provide more information or ask about the status of your request.");
			$("#info").toggleClass("ui positive message");
			$("#info").toggleClass("ui negative message");
			$("#infotable").hide();
			$("#warning").hide();
		}
	});
}
		</script>
		<title>GIN-DOI</title>

		<meta name="theme-color" content="#ffffff">


	</head>
	<body>
		<div class="full-height" id="main">
			<div class="following bar light">
				<div class="ui container">
					<div class="ui grid">
						<div class="column">
							<div class="ui top secondary menu">
								<a class="item brand" href="https://gin.g-node.org/">
									<img class="ui mini image" src="/assets/img/favicon.png">
								</a>
								<a class="item active" href="https://gin.g-node.org/{{.Repository}}">Back to GIN</a>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div>
				<div class="ui vertically padded grid head">
					<div class="column center">
						<h1>Welcome to the GIN DOI service
							<i class="mega-octicon octicon octicon-squirrel"></i>
						</h1>
					</div>
				</div>

				<div class="ui container">
					{{if .DOIInfo.Title}}
						<div class="ui positive message" id="info">
							<div>
								Your repository "{{.DOIInfo.Title}}" fulfills all necessary requirements!
								Click the button below to start the DOI request.
							</div>
						</div>
						<div class="ui info message" id="infotable">
							<div id="infobox">The following information will be registered with your DOI request.
								It will also be presented alongside the data. Please check it thoroughly and modify your datacite file
								if any information is wrong or missing.
							</div>
							<hr>
							<div id="cloudberry-view" class="tab-size-8">
								<table class="ui fixed single line table">
									<thead>
										<tr>
											<th class="three wide">
											</th>
											<th class="fourteen wide">
											</th>
										</tr>
									</thead>
									<tbody>
										<tr>
											<td>Title</td>
											<td>{{.DOIInfo.Title}}</td>
										</tr>
										<tr>
											<td>Authors</td>
											<td>
												{{range $index, $auth := .DOIInfo.Authors}}
													{{ $auth.LastName }},{{ $auth.FirstName }}; {{ $auth.Affiliation}}; {{$auth.ID}}
													<br>
												{{end}}
											</td>
										</tr>
										{{if .DOIInfo.Description}}
											<tr>
												<td>Description</td>
												<td>{{.DOIInfo.Description}}
												</td>
											</tr>
										{{end}}
										{{if .DOIInfo.License}}
											<tr>
												<td>License</td>
												<td>{{.DOIInfo.License.Name}} ({{.DOIInfo.License.URL}})
												</td>
											</tr>
										{{end}}
										<tr>
											<td>References</td>
											<td>
												{{range $index, $ref := .DOIInfo.References}}
													{{ $ref.Name }} [{{ $ref.ID }}] ({{ $ref.Reftype }})
													<br>
												{{end}}
											</td>
										</tr>
										<tr>
											<td>Funding</td>
											<td>
												{{range $index, $ref := .DOIInfo.Funding}}
													{{ $ref}}
													<br>
												{{end}}
											</td>
										</tr>
										{{if .DOIInfo.Keywords}}
											<tr>
												<td>Keywords</td>
												<td>
													{{range $index, $sub := .DOIInfo.Keywords}}
														{{ $sub }}
														<br>
													{{end}}
												</td>
											</tr>
										{{end}}
										<tr>
											<td>Resource Type</td>
											<td>
												<i>{{.DOIInfo.GetType}}</i><br>
											</td>
										</tr>
									</tbody>
								</table>
							</div>
							<div class="ui negative icon message" id="warning">
								<i class="warning icon"></i>
								<div class="content">
									<div class="header">Please thoroughly check the following before proceeding</div>
									<ul align="left">
										<li>Did you upload all data?</li>
										<li>Does your repository contain a LICENSE file?</li>
										<li>Does the license in the LICENSE file match the license you provided in datacite.yml?</li>
										<li>Does your repository contain a good description of the data?</li>
									</ul>
									<p><b>Please be aware that all data in your repository will be part of the archived file that will be used for the DOI registration.</b></p>
									Please make sure it does not contain any private files, SSH keys, address books, password collections, or similar sensitive, private data.
									<p><b>All files and data in the repository will be part of the public archive!</b></p>
								</div>
							</div>
							<div class="ui primary button" id="doify">Request DOI Now</div>
						{{else}}
							<div class="ui warning message">
								<div><b>DOI request failed</b>
									<p>{{.Message}}</p>
								</div>
							</div>
						{{end}}
						</div>
				</div>
			</div>
		</div>
		<footer>
			<div class="ui container">
				<div class="ui center links item brand footertext">
					<a href="http://www.g-node.org"><img class="ui mini footericon" src="https://projects.g-node.org/assets/gnode-bootstrap-theme/1.2.0-snapshot/img/gnode-icon-50x50-transparent.png"/>© G-Node, 2016-2019</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/about">About</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/imprint">Imprint</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/contact">Contact</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/Terms+of+Use">Terms of Use</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/Datenschutz">Datenschutz</a>

				</div>
				<div class="ui center links item brand footertext">
					<span>Powered by:      <a href="https://github.com/gogs/gogs"><img class="ui mini footericon" src="/assets/img/gogs.svg"/></a>         </span>
					<span>Hosted by:       <a href="https://neuro.bio.lmu.de"><img class="ui mini footericon" src="/assets/img/lmu.png"/></a>          </span>
					<span>Funded by:       <a href="https://www.bmbf.de"><img class="ui mini footericon" src="/assets/img/bmbf.png"/></a>         </span>
					<span>Registered with: <a href="https://doi.org/10.17616/R3SX9N"><img class="ui mini footericon" src="/assets/img/re3.png"/></a>          </span>
					<span>Recommended by:  <a href="https://www.nature.com/sdata/policies/repositories#neurosci"><img class="ui mini footericon" src="/assets/img/sdatarecbadge.jpg"/><a href="https://journals.plos.org/plosone/s/data-availability#loc-neuroscience"><img class="ui mini footericon" src="/assets/img/sm_plos-logo-sm.png"/></a></span>
				</div>
			</div>
		</footer>

	</body>
</html>`

const landingPageTmpl = `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<link rel="stylesheet" href="/assets/css/semantic-2.3.1.min.css">
		<link rel="stylesheet" href="/assets/css/gogs.css">
		<link rel="stylesheet" href="/assets/css/custom.css">
		<title>G-Node GIN-DOI</title>
	</head>
	<body>
		<div class="following bar light">
			<div class="ui container">
				<div class="ui grid">
					<div class="column">
						<div class="ui top secondary menu">
							<a class="item brand" href="https://gin.g-node.org/">
								<img class="ui mini image" src="/assets/img/favicon.png">
							</a>
							<a class="item" href="https://doid.gin.g-node.org/">Home</a>
							<a class="item active" href="">Data</a>
						</div>
					</div>
				</div>
			</div>
		</div>

		<div class="ui stackable middle very relaxed page grid">
			<div class="sixteen wide center aligned centered column">
				<h1>{{.DOIInfo.Title}}</h1>
				<p><b>Dataset</b></p>
				<table class="ui very basic table">
					<tbody>
						<tr>
							<td>Title</td>
							<td>{{.DOIInfo.Title}}</td>
						</tr>
						<tr>
							<td>Authors</td>
							<td>
								{{range $index, $auth := .DOIInfo.Authors}}
									{{ $auth.LastName }} {{ $auth.FirstName }},
								{{end}}
							</td>
						</tr>
						{{if .DOIInfo.Description}}
							<tr>
								<td>Description</td>
								<td>{{.DOIInfo.Description}}
								</td>
							</tr>
						{{end}}
						{{if .DOIInfo.License}}
							<tr>
								<td>License</td>
								<td>{{.DOIInfo.License.Name}} (<a href="{{.DOIInfo.License.URL}}">{{.DOIInfo.License.URL}}</a>)
								</td>
							</tr>
						{{end}}
						<tr>
							<td>References</td>
							<td>
								{{range $index, $ref := .DOIInfo.References}}
									{{ $ref.Name }}
									{{if $ref.ID}}
										[<a href={{$ref.GetURL}}>{{$ref.ID}}</a>]
									{{end}}
									<br>
								{{end}}
							</td>
						</tr>
						<tr>
							<td>Funding</td>
							<td>
								{{range $index, $ref := .DOIInfo.Funding}}
									{{ $ref}}
									<br>
								{{end}}
							</td>
						</tr>
						{{if .DOIInfo.Keywords}}
							<tr>
								<td>Keywords</td>
								<td>
									{{range $index, $sub := .DOIInfo.Keywords}}
										{{ $sub }}
										<br>
									{{end}}
								</td>
							</tr>
						{{end}}
						<tr>
							<td>Data</td>
							<td>
								This dataset can be browsed online <a href="https://gin.g-node.org/{{.GetDOIURI}}">here</a> or downloaded as a
								<a href="{{.DOIInfo.UUID}}.zip">zip archive ({{.DOIInfo.FileSize}})</a>.
								The current version of the dataset repository, possibly with updates, can be found <a href="https://gin.g-node.org/{{.Repository}}">here</a>.
							</td>
						</tr>
						<tr>
							<td>DOI</td>
							<td><a href="#" class ="ui grey label">{{.DOIInfo.DOI}}</a></td>
						</tr>
						<tr>
							<td>Citation</td>
							<td>
								<i>This dataset can be cited as:</i><br>
								{{.DOIInfo.GetCitation}}<br>
								<i>Please also consider citing the material listed in the references</i>
							</td>
						</tr>
						<tr>
							<td>Resource Type</td>
							<td>
								<i>{{.DOIInfo.GetType}}</i><br>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>
		<footer>
			<div class="ui container">
				<div class="ui center links item brand footertext">
					<a href="http://www.g-node.org"><img class="ui mini footericon" src="https://projects.g-node.org/assets/gnode-bootstrap-theme/1.2.0-snapshot/img/gnode-icon-50x50-transparent.png"/>© G-Node, 2016-2018</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/about">About</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/imprint">Imprint</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/contact">Contact</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/Terms+of+Use">Terms of Use</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/Datenschutz">Datenschutz</a>
				</div>
				<div class="ui center links item brand footertext">
					<span>Powered by:      <a href="https://github.com/gogits/gogs"><img class="ui mini footericon" src="/assets/img/gogs.svg"/></a>         </span>
					<span>Hosted by:       <a href="http://neuro.bio.lmu.de"><img class="ui mini footericon" src="/assets/img/lmu.png"/></a>          </span>
					<span>Funded by:       <a href="http://www.bmbf.de"><img class="ui mini footericon" src="/assets/img/bmbf.png"/></a>         </span>
					<span>Registered with: <a href="http://doi.org/10.17616/R3SX9N"><img class="ui mini footericon" src="/assets/img/re3.png"/></a>          </span>
					<span>Recommended by:  <a href="https://www.nature.com/sdata/policies/repositories#neurosci"><img class="ui mini footericon" src="/assets/img/sdatarecbadge.jpg"/><a href="https://journals.plos.org/plosone/s/data-availability#loc-neuroscience"><img class="ui mini footericon" src="/assets/img/sm_plos-logo-sm.png"/></a></span>
				</div>
			</div>
		</footer>
	</body>
</html>`

const doiXML = `<?xml version="1.0" encoding="UTF-8"?>
<resource xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://datacite.org/schema/kernel-4" xsi:schemaLocation="http://datacite.org/schema/kernel-4 http://schema.datacite.org/meta/kernel-4/metadata.xsd">
  <identifier identifierType="DOI">{{.DOI}}</identifier>
  <creators>{{range $index, $auth := .Authors}}
    <creator>
      <creatorName>{{$.EscXML $auth.LastName}}, {{$.EscXML $auth.FirstName}}</creatorName>
        {{if $auth.GetValidID}}<nameIdentifier schemeURI="{{$auth.GetValidID.URI}}" nameIdentifierScheme="{{$auth.GetValidID.Scheme}}">{{$.EscXML $auth.GetValidID.ID}}</nameIdentifier>{{end}}
        {{if $auth.Affiliation}}<affiliation>{{$.EscXML $auth.Affiliation}}</affiliation>{{end}}
    </creator>{{end}}
  </creators>
  <titles>
    <title>{{.EscXML .Title}}</title>
  </titles>
  {{if .Description}}<descriptions>
    <description descriptionType="Abstract">
    {{.EscXML  .Description}}
    </description>
  </descriptions>{{end}}
  {{if .License}}<rightsList>
    <rights {{if .License.URL}}rightsURI="{{.License.URL}}"{{end}}> {{$.EscXML .License.Name}}</rights>
  </rightsList>{{end}}
  {{if .Keywords}}<subjects>{{range $index, $kw := .Keywords}}
     <subject>{{$.EscXML $kw}}</subject>{{end}}
  </subjects>{{end}}
  {{if .References}}<relatedIdentifiers>{{range $index, $ref := .References}}
  <relatedIdentifier relatedIdentifierType="Handle" relationType="{{$ref.Reftype}}">{{$.EscXML $ref.Name}}</relatedIdentifier>{{end}}
  </relatedIdentifiers>{{end}}
  {{if .Funding}}<fundingReferences>{{range $index, $fu := .Funding}}
  <fundingReference><funderName>{{$.EscXML $fu}}</funderName></fundingReference>{{end}}
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
