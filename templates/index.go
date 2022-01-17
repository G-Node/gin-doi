package gdtmpl

// IndexPage is the template for rendering the index page of registered datasets.
const IndexPage = `<!DOCTYPE html>
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

		<title>G-Node Open Data</title>
	</head>
	<body>
		{{ template "Nav" }}
		<div class="ui stackable middle very relaxed page grid">
			<div class="sixteenn wide center aligned centered column">
				<h1>G-Node Open Data</h1>
				<p><b>Registered Datasets</b></p>
				<table class="ui very basic table">
					<thead>
						<tr>
							<th class="ten wide">Title</th>
							<th class="two wide">Date</th>
							<th class="four wide">DOI</th>
						</tr>
					</thead>
					<tbody>
						{{ range . }}
							<tr>
								<td><a href="https://doi.org/{{ .Shorthash }}">{{ .Title }}</a>
								<br>{{ .Authors }}</td>
								<td>{{ .Isodate }}</td>
								<td><a href="https://doi.org/{{ .Shorthash }}" class ="ui grey label">{{ .Shorthash }}</a></td>
							</tr>
						{{end}}
					</tbody>
				</table>
				More public datasets can be found at <a href="https://gin.g-node.org">gin.g-node.org</a>
			</div>
		</div>
		{{ template "Footer" }}
	</body>
</html>`
