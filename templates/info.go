package gdtmpl

// DOIInfo is a partial template for the rendering of all the DOI info.  The
// template is used for the generation of the landing page as well as the
// preparation page (before submitting a request) and the preview on the
// repository front page on GIN.
const DOIInfo = `
<div class="doi title">
	<h2>{{.ResourceType.Value}}</h2>
	<h1 itemprop="name">{{index .Titles 0}}</h1>
	{{AuthorBlock .Creators}}
	<meta itemprop="identifier" content="doi:{{.Identifier.ID}}">
	<p>
	<a href="{{if .Identifier.ID}}https://doi.org/{{.Identifier.ID}}{{end}}" class="ui black doi label" itemprop="url">DOI: {{if .Identifier.ID}}{{.Identifier.ID}}{{else}}UNPUBLISHED{{end}}</a>
	<a href="{{if .SourceRepository}}https://gin.g-node.org/{{.SourceRepository}}{{end}}" class="ui blue doi label" data-tooltip="Browse the live dataset's contents on GIN. The repository may contain updates."><i class="doi label octicon octicon-link"></i>&nbsp;BROWSE REPOSITORY</a>
	<a href="{{if .ForkRepository}}https://gin.g-node.org/{{.ForkRepository}}{{end}}" class="ui blue doi label" data-tooltip="Browse the archived dataset's contents on GIN. This is a snapshot of the published version."><i class="doi label octicon octicon-link"></i>&nbsp;BROWSE ARCHIVE</a>
	<a href="{{if .Identifier.ID}}{{Replace .Identifier.ID "/" "_"}}.zip{{end}}" class="ui green doi label"><i class="doi label octicon octicon-desktop-download"></i>&nbsp;DOWNLOAD ARCHIVE (ZIP{{if .Sizes}} {{index .Sizes 0}}{{end}})</a>
	</p>
	<p><strong>Published</strong> {{FormatIssuedDate .}} | <strong>License</strong> {{with index .RightsList 0}} <a href="{{.URL}}" itemprop="license">{{.Name}}</a>{{end}}</p>
</div>
<hr>

{{NewVersionNotice .}}

{{if .Descriptions}}
	<h3>Description</h3>
	<p itemprop="description">{{with index .Descriptions 0}}{{.Content}}{{end}}</p>
{{end}}

{{if .Subjects}}
	<h3>Keywords</h3>
	| {{range $index, $kw := .Subjects}} <a href="/keywords/{{$kw | KeywordPath}}/">{{$kw}}</a> | {{end}}
	<meta itemprop="keywords" content="{{JoinComma .Subjects}}">
{{end}}

{{with $refs := FormatReferences .}}
{{if $refs}}
	<h3>References</h3>
	<ul class="doi itemlist">
		{{range $index, $ref := $refs}}
			<li itemprop="citation" itemscope itemtype="http://schema.org/CreativeWork"><span itemprop="name">{{$ref.Name}} {{$ref.Citation}}</span>{{if $ref.ID}} <a href="{{$ref.GetURL}}" itemprop="url"><span itemprop="identifier">{{$ref.GetURL}}</span></a>{{end}}</li>
		{{end}}
	</ul>
{{end}}
{{end}}

{{if .FundingReferences}}
	<h3>Funding</h3>
	<ul class="doi itemlist">
		{{range $index, $funding := .FundingReferences}}
			<li itemprop="funder" itemscope itemtype="http://schema.org/Organization"><span itemprop="name">{{$funding.Funder}}</span> {{$funding.AwardNumber}}</li>
		{{end}}
	</ul>
{{end}}

{{OldVersionLink .}}
`
