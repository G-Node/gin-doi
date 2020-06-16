package gdtmpl

const KeywordIndex = `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<link rel="stylesheet" href="/assets/css/semantic-2.3.1.min.css">
		<link rel="stylesheet" href="/assets/octicons-4.3.0/octicons.min.css">
		<link rel="stylesheet" href="/assets/css/gogs.css">
		<link rel="stylesheet" href="/assets/css/custom.css">
		<title>G-Node Open Data: Keywords</title>
	</head>
	<body>
		<div class="full height">
			{{template "Nav"}}
			<div class="home middle very relaxed page grid" id="main">
				<div class="six center aligned centered column">
					<h1>G-Node Open Data</h1>
					<h2>Keywords</h2>
				</div>

			<div class="ui four column stackable grid container">
				{{range $idx, $keyword := .KeywordList}}
					<div class="column"><div class="ui"><a class="text bold" href="{{$keyword}}">{{$keyword}}</a> <span class="right">{{index $.KeywordMap $keyword | len}}</span></div></div>
				{{end}}
			</div>
			</div>
		</div>
		{{template "Footer"}}
	</body>
</html>`
