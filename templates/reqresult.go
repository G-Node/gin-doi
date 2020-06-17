package gdtmpl

// RequestResult is the template for rendering success or failure after the
// user submits a request through the RequestPage.
const RequestResult = `<!DOCTYPE html>
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
		<div class="full height" id="main">
			<div class="following bar light">
				<div class="ui container">
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
			<div class="home middle very relaxed page" id="main">
				<div class="ui container">
					<div class="ui {{.Level}} message">
				{{if .Success}}
						<div><b>DOI request submitted</b></div>
				{{else}}
						<div><b>DOI request failed</b></div>
				{{end}}
					</div>
					<div class="ui info message">
						{{.Message}}
					</div>
				</div>
			</div>
		</div>
		{{template "Footer"}}
	</body>
</html>`
