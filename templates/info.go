package gdtmpl

// DOIInfo is a partial template for the rendering of all the DOI info.  The
// template is used for the generation of the landing page as well as the
// preparation page (before submitting a request) and the preview on the
// repository front page on GIN.
const DOIInfo = `
<div class="doi title">
	<h2>{{.Metadata.ResourceType.Value}}</h2>
	<h1 itemprop="name">{{index .Metadata.Titles 0}}</h1>
	{{AuthorBlock .Metadata.Creators}}
	<meta itemprop="identifier" content="doi:{{.Metadata.DOI}}">
	<p>
	<a href="https://doi.org/{{.Metadata.DOI}}" class="ui black doi label" itemprop="url">DOI: {{if .Metadata.DOI}}{{.Metadata.DOI}}{{else}}UNPUBLISHED{{end}}</a>
	<a href="" class="ui blue doi label"><i class="doi label octicon octicon-link"></i>&nbsp;BROWSE REPOSITORY (fix URL)</a>
	<a href="" class="ui blue doi label"><i class="doi label octicon octicon-link"></i>&nbsp;BROWSE ARCHIVE (fix URL)</a>
	<a href="" class="ui green doi label"><i class="doi label octicon octicon-desktop-download"></i>&nbsp;DOWNLOAD {{.Metadata.ResourceType.Value | Upper}} ARCHIVE (ZIP{{if .Metadata.Size}} {{.Metadata.Size}}{{end}}) (fix URL)</a>
	</p>
	<p><strong>Published</strong> {{.Metadata.DateTime.Format "02 Jan. 2006"}} | <strong>License</strong> {{with index .Metadata.RightsList 0}} <a href="{{.URL}}" itemprop="license">{{.Name}}</a>{{end}}</p>
</div>
<hr>

{{if .Metadata.Descriptions}}
	<h3>Description</h3>
	<p itemprop="description">{{with index .Metadata.Descriptions 0}}{{.Content}}{{end}}</p>
{{end}}

{{if .Metadata.Subjects}}
	<h3>Keywords</h3>
	| {{range $index, $kw := .Metadata.Subjects}} <a href="/keywords/{{$kw}}">{{$kw}}</a> | {{end}}
	<meta itemprop="keywords" content="{{JoinComma .Metadata.Subjects}}">
{{end}}

{{if .Metadata.RelatedIdentifiers}}
	<h3>References</h3>
	<ul class="doi itemlist">
		{{range $index, $ref := .Metadata.RelatedIdentifiers}}
			<li itemprop="citation" itemscope itemtype="http://schema.org/CreativeWork"><span itemprop="name">{$ref.Name} {$ref.Citation}</span>{if $ref.ID} <a href={$ref.GetURL} itemprop="url"><span itemprop="identifier">{$ref.ID}</span></a>{end}</li>
		{{end}}
	</ul>
{{end}}

{{if .Metadata.FundingReferences}}
	<h3>Funding</h3>
	<ul class="doi itemlist">
		{{range $index, $funding := .Metadata.FundingReferences}}
			<li itemprop="funder" itemscope itemtype="http://schema.org/Organization"><span itemprop="name">{{$funding.Funder}}</span> {{$funding.AwardNumber}}</li>
		{{end}}
	</ul>
{{end}}
`
