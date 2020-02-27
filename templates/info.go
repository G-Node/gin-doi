package gdtmpl

// DOIInfo is a partial template for the rendering of all the DOI info.  The
// template is used for the generation of the landing page as well as the
// preparation page (before submitting a request) and the preview on the
// repository front page on GIN.
const DOIInfo = `
<div class="doi title">
	<h2>{{.DOIInfo.ResourceType}}</h2>
	<h1 itemprop="name">{{.DOIInfo.Title}}</h1>
	{{AuthorBlock .DOIInfo.Authors}}
	{{if .DOIInfo.DOI}}
		<meta itemprop="identifier" content="doi:{{.DOIInfo.DOI}}">
		<p>
		<a href="https://doi.org/{{.DOIInfo.DOI}}" class="ui black doi label" itemprop="url">DOI: {{.DOIInfo.DOI}}</a>
		<a href="https://gin.g-node.org/{{.Repository}}" class="ui blue doi label"><i class="doi label octicon octicon-link"></i>&nbsp;BROWSE REPOSITORY</a>
		<a href="https://gin.g-node.org/{{.GetDOIURI}}" class="ui blue doi label"><i class="doi label octicon octicon-link"></i>&nbsp;BROWSE ARCHIVE</a>
		<a href="{{.DOIInfo.FileName}}" class="ui green doi label"><i class="doi label octicon octicon-desktop-download"></i>&nbsp;DOWNLOAD {{.DOIInfo.ResourceType | Upper}} ARCHIVE (ZIP{{if .DOIInfo.FileSize}} {{.DOIInfo.FileSize}}{{end}})</a>
		</p>
	{{end}}
	<p><strong>Published</strong> {{.DOIInfo.PrettyDate}} | <strong>License</strong> <a href="{{.DOIInfo.License.URL}}" itemprop="license">{{.DOIInfo.License.Name}}</a></p>
</div>
<hr>

{{if .DOIInfo.Description}}
	<h3>Description</h3>
	<p itemprop="description">{{.DOIInfo.Description}}</p>
{{end}}

{{if .DOIInfo.Keywords}}
	<h3>Keywords</h3>
	| {{range $index, $kw := .DOIInfo.Keywords}} <a href="/keywords/{{$kw}}">{{$kw}}</a> | {{end}}
	<meta itemprop="keywords" content="{{JoinComma .DOIInfo.Keywords}}">
{{end}}


{{if .DOIInfo.References}}
	<h3>References</h3>
	<ul class="doi itemlist">
		{{range $index, $ref := .DOIInfo.References}}
			<li itemprop="citation" itemscope itemtype="http://schema.org/CreativeWork"><span itemprop="name">{{$ref.Name}} {{$ref.Citation}}</span>{{if $ref.ID}} <a href={{$ref.GetURL}} itemprop="url"><span itemprop="identifier">{{$ref.ID}}</span></a>{{end}}</li>
		{{end}}
	</ul>
{{end}}

{{if .DOIInfo.Funding}}
	<h3>Funding</h3>
	<ul class="doi itemlist">
		{{range $index, $org := .DOIInfo.Funding}}
			<li itemprop="funder" itemscope itemtype="http://schema.org/Organization"><span itemprop="name">{{FunderName $org}}</span> {{AwardNumber $org}}</li>
		{{end}}
	</ul>
{{end}}


<h3>Citation</h3>
<i>This dataset can be cited as:</i><br>
{{.DOIInfo.GetCitation}}<br>
`
