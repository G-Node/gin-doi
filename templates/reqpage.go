package gdtmpl

// RequestPage is the template for rendering the request page where the
// user can see the metadata they are providing and finalise their registration
// request.
const RequestPage = `<!DOCTYPE html>
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

		<title>G-Node DOI</title>

		<meta name="theme-color" content="#ffffff">
	</head>
	<body>
		<div class="full height">
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
			<div class="home middle very relaxed page grid" id="main">
				<div class="ui container wide centered column doi">
					<div class="column center">
						<h1>Welcome to the GIN DOI service <i class="mega-octicon octicon octicon-squirrel"></i></h1>
					</div>
					<div class="ui info message" id="infotable">
						<div id="infobox">
							The following <strong>preview</strong> shows the information that will be published in the DOI registry and will be presented permanently alongside the data in your repository.
							Please review it carefully before clicking the Request DOI button.
							If anything needs to be changed use the Cancel button to return to your repository and edit the datacite.yml file.
						</div>
					</div>
					<hr>
					{{template "doiInfo" .Metadata}}
					<hr>
					<div class="column center">
						<h3>END OF PREVIEW</h3>
					</div>
					<div class="ui negative icon message" id="warning">
						<i class="warning icon"></i>
						<div class="content">
							<div class="header">Please thoroughly check the following before proceeding</div>
							<ul align="left">
								<li>Did you upload all data?</li>
								<li>Does your repository contain a comprehensive description of the data (preferably in the README.md file)?</li>
								<li>Does your repository contain a LICENSE file with a license matching the one indicated in datacite.yml?</li>
							</ul>
							<p><b>Please be aware that the entire repository will be published.</b></p>
							<p><b>Please make sure it does not contain any private files, SSH keys, address books, password collections, or similar sensitive, private data.</b></p>
							<p><b>All contents of the repository will be part of the public archive!</b></p>
						</div>
					</div>
					<form action="/submit" method="post">
						<input type="hidden" id="reqdata" name="reqdata" value="{{.EncryptedRequestData}}">
						<div class="column center">
							<a class="ui black button" href=https://gin.g-node.org/{{.Repository}}>Cancel</a>
							<button class="ui primary button" type="submit">Request DOI Now</button>
						</div>
					</form>
				</div>
			</div>
		</div>
		<footer>
			<div class="ui container">
				<div class="ui center links item brand footertext">
					<a href="http://www.g-node.org"><img class="ui mini footericon" src="https://projects.g-node.org/assets/gnode-bootstrap-theme/1.2.0-snapshot/img/gnode-icon-50x50-transparent.png"/>© G-Node, 2016-2020</a>
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
