package gdtmpl

// Nav is the template for the nav bar at the top of every page
const Nav = `
			<div class="following bar light">
				<div class="ui container">
					<div class="ui grid">
						<div class="column">
							<div class="ui top secondary menu">
								<a class="item brand" href="https://gin.g-node.org/">
									<img class="ui mini image" src="/assets/img/favicon.png">
								</a>
								<a class="item" href="/">Published Data</a>
								<a class="item" href="/keywords">Keywords</a>
								<a class="item" href="https://gin.g-node.org/explore/repos">Public datasets on GIN</a>
							</div>
						</div>
					</div>
				</div>
			</div>
`

// Footer is the template for every page footer
const Footer = `
		<footer>
			<div class="ui container">
				<div class="ui center links item brand footertext">
					<a href="http://www.g-node.org"><img class="ui mini footericon" src="https://projects.g-node.org/assets/gnode-bootstrap-theme/1.2.0-snapshot/img/gnode-icon-50x50-transparent.png"/>© G-Node, 2016-2022</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/about">About</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/imprint">Imprint</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/contact">Contact</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/Terms+of+Use">Terms of Use</a>
					<a href="https://gin.g-node.org/G-Node/Info/wiki/Datenschutz">Datenschutz</a>
				</div>
				<div class="ui center links item brand footertext">
					<span>Powered by:      <a href="https://github.com/gogits/gogs"><img class="ui mini footericon" src="/assets/img/gogs.svg"/></a>         </span>
					<span>Hosted by:       <a href="http://neuro.bio.lmu.de"><img class="ui mini footericon" src="/assets/img/lmu.png"/></a>          </span>
					<span>Funded by:       <a href="http://www.bmbf.de"><img class="ui mini footericon" src="/assets/img/bmbf.png"/></a>         </span>
					<span>Registered with: <a href="http://doi.org/10.17616/R3SX9N"><img class="ui mini footericon" src="/assets/img/re3data_logo.png"/></a>          </span>
					<span>Recommended by:  
						<a href="https://www.nature.com/sdata/policies/repositories#neurosci"><img class="ui mini footericon" src="/assets/img/sdatarecbadge.jpg"/></a>
						<a href="https://fairsharing.org/recommendation/PLOS"><img class="ui mini footericon" src="/assets/img/sm_plos-logo-sm.png"/></a>
						<a href="https://fairsharing.org/recommendation/eLifeRecommendedRepositoriesandStandards"><img class="ui mini footericon" src="/assets/img/elife-logo-xs.fd623d00.svg"/></a>
					</span>
				</div>
			</div>
		</footer>
`
