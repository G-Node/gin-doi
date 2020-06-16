package gdtmpl

// LandingPage is the template for rendering the landing page of the registered dataset.
const LandingPage = `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<link rel="stylesheet" href="/assets/css/semantic-2.3.1.min.css">
		<link rel="stylesheet" href="/assets/octicons-4.3.0/octicons.min.css">
		<link rel="stylesheet" href="/assets/css/gogs.css">
		<link rel="stylesheet" href="/assets/css/custom.css">
		<title>G-Node Open Data: {{index .Titles 0}}</title>
	</head>
	<body>
		<div class="full height">
			{{template "Nav"}}
			<div class="home middle very relaxed page grid" id="main">
				<div class="ui container sixteen wide centered column doi">
					<span itemscope itemtype="http://schema.org/Dataset">
						{{template "DOIInfo" .}}
						<h3>Citation</h3>
						{{FormatCitation .}}<br>
					</span>
				</div>
			</div>
		</div>
		{{template "Footer"}}
	</body>
</html>`
