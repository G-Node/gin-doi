package gdtmpl

// Keyword is the template for an HTML page
// linking DOI pages featuring a specifc keyword.
const Keyword = `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1">

		<link rel="shortcut icon" href="/assets/img/favicon.png">
		<link rel="stylesheet" href="/assets/css/semantic-2.3.1.min.css">
		<link rel="stylesheet" href="/assets/octicons-4.3.0/octicons.min.css">
		<link rel="stylesheet" href="/assets/css/gogs.css">
		<link rel="stylesheet" href="/assets/css/custom.css">

		<title>G-Node Open Data: {{.Keyword}}</title>
	</head>
	<body>
		<div class="full height">
			{{template "Nav"}}
			<div class="home middle very relaxed page grid" id="main">
				<div class="sixteen wide center aligned centered column">
					<h1>G-Node Open Data</h1>
					{{$n := len .Datasets}}
					<h2>{{$n}} Registered Dataset{{if gt $n 1}}s{{end}} with keyword: {{.Keyword}}</h2>
				</div>
				<div class="ui container sixteen wide centered column doi">
					<table class="ui very basic table">
						<thead><tr> <th class="ten wide"></th><th class="two wide"></th> <th class="four wide"></th></tr></thead>
						{{range $idx, $dataset := .Datasets}}
							{{$title := index $dataset.Titles 0}}
							{{$date := FormatIssuedDate $dataset}}
							{{$doi := $dataset.Identifier.ID}}
							{{$authors := FormatAuthorList $dataset}}
							<tr><td><a href=https://doi.org/{{$doi}}>{{$title}}</a><br>{{$authors}}</td><td>{{$date}}</td> <td><a href=https://doi.org/{{$doi}}>{{$doi}}</a></td></tr>
						{{end}}
					</table>
				</div>
			</div>
		</div>
		{{template "Footer"}}
	</body>
</html>`
