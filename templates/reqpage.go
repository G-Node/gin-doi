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
								<a class="item brand" href="{{GINServerURL}}/">
									<img class="ui mini image" src="/assets/img/favicon.png">
								</a>
								<a class="item active" href="{{GINServerURL}}/{{.Repository}}">Back to GIN</a>
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

					{{if HasGitModules GINServerURL .Repository}}
					<div class="ui negative message" id="gitmodulewarning">
						<div id="gitmodulebox">
							<div class="header">Your repository appears to contain git submodules</div>
							<p><b>Please note that content linked via git submodules will not be included in the published dataset.</b></p>
						</div>
					</div>
					{{end}}

					<div class="ui info message" id="infotable">
						<div id="infobox">
							The following <strong>preview</strong> shows the information that will be published in the DOI registry and will be presented permanently alongside the data in your repository.
							Please review it carefully before clicking the Request DOI button.
							If anything needs to be changed use the Cancel button to return to your repository and edit the datacite.yml file.
						</div>
					</div>
					<hr>
					{{template "DOIInfo" .Metadata}}
					<hr>
					<div class="column center">
						<h3>END OF PREVIEW</h3>
					</div>
					<div class="ui negative icon message" id="warning">
						<i class="warning icon"></i>
						<div class="content">
							<div class="header">Please thoroughly check the following before proceeding</div>
							<div class="content" align="left">
								<ul>
									<li>Did you upload all data?</li>
									<li>Does your repository contain a comprehensive description of the data (preferably in the README.md file)?</li>
									<li>Does your repository contain a LICENSE file with a license matching the one indicated in datacite.yml?</li>
									<li>Does your repository contain code or content licensed under different terms? Please include a separate LICENSE file for parts of the repository and describe its application in the README.</li>
								</ul>
							</div>
							<p><b>Please be aware that the entire repository will be published.</b></p>
							<p><b>Please note that content linked via git submodules will not be included in the published dataset.</b></p>
							<p><b>Please make sure it does not contain any private files, SSH keys, address books, password collections, or similar sensitive, private data.</b></p>
							<p><b>All contents of the repository will be part of the public archive!</b></p>
						</div>
					</div>
					<form action="/submit" method="post">
						<input type="hidden" id="reqdata" name="reqdata" value="{{.EncryptedRequestData}}">
						<div class="column center">
							<a class="ui button" href={{GINServerURL}}/{{.Repository}}>Cancel</a>
							<button class="ui green button" type="submit">Request DOI Now</button>
						</div>
					</form>
				</div>
			</div>
		</div>
		{{template "Footer"}}
	</body>
</html>`
