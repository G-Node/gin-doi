package main

const (
	msgInvalidRequest    = `Invalid request data received.  Please note that requests should only be submitted through repository pages on <a href="https://gin.g-node.org">GIN</a>.  If you followed the instructions in the <a href="https://gin.g-node.org/G-Node/Info/wiki/DOIfile">DOI registration guide</a> and arrived at this error page, please <a href="mailto:gin@g-node.org">contact us</a> for assistance.`
	msgInvalidDOI        = `The DOI file is missing in the <b>master</b> branch or not valid.<br>See the messages below for specific issues with the provided data.<br>Also, please see <a href="https://gin.g-node.org/G-Node/Info/wiki/DOIfile">the DOI guide</a> for detailed instructions.`
	msgInvalidURI        = "Please provide a valid repository URI"
	msgAlreadyRegistered = `<div class="content">
								<div class="header"> A DOI is already registered for your dataset.</div>
								Your DOI is: <br>
								<div class ="ui label label-default"><a href="https://doi.org/%s">%s</a></div></br>
								If this is incorrect or you would like to register a new version of your dataset, please <a href=mailto:gin@g-node.org>contact us</a>.
							</div>`
	msgServerIsArchiving = `<div class="content">
			<div class="header">The DOI server has started archiving your repository.</div>
		We have reserved the following DOI for your dataset:<br>
		<div class="ui label label-default">%s</div><br>
		An email has been sent containing the above information to your registered address on GIN.<br>
		Please note that the registration process includes a manual curation step. It may therefore take up to two work days until the DOI is available. If any changes to the repository should be necessary you will be contacted by the curation team.<br>
		We will notify you via email once the process is finished.<br>
		<div class="ui tabs divider"> </div>
		<b>This page can safely be closed. You do not need to keep it open.</b>
		</div>
		`
	msgSubmitSuccessEmail = `Dear %s,

We have received your request to publish the GIN repository %s.
The following DOI has been reserved: %s

Please note that the registration process includes a manual curation step. It may therefore take up to two work days until the DOI is available. If any changes to the repository should be necessary you will be contacted by the curation team.
We will notify you via email once the process is finished.

If you would like to make any changes to the dataset before it is published, or if you have any questions or concerns, feel free to contact us at gin@g-node.org.
`
	msgNotLoggedIn      = `You are not logged in with the gin service. Login <a href="http://gin.g-node.org/">here</a>`
	msgNoToken          = "No authentication token provided"
	msgNoUser           = "No username provided"
	msgNoTitle          = `No <b>title</b> provided.`
	msgNoAuthors        = `No <b>authors</b> provided.`
	msgInvalidAuthors   = "Not all authors valid. Please provide at least a last name and a first name."
	msgNoDescription    = `No <b>description</b> provided.`
	msgNoMaster         = "Could not access the repository <b>master</b> branch. DOI requests require the master branch of the requesting repository."
	msgNoLicense        = `No valid <b>license</b> provided. Please specify a license URL and name and make sure it matches the license file in the repository.`
	msgNoLicenseFile    = `The LICENSE file is missing in the required <b>master</b> branch. The full text of the license is required to be in the repository when publishing.<br>See the <a href="https://gin.g-node.org/G-Node/Info/wiki/Licensing">Licensing</a> help page for details and links to recommended data licenses.`
	msgLicenseMismatch  = `The LICENSE file does not match the license specified in the metadata. See the <a href="https://gin.g-node.org/G-Node/Info/wiki/Licensing">Licensing</a> help page for links to full text for available licenses.`
	msgInvalidReference = `Not all <b>Reference</b> entries are valid. Please provide the full citation and type of the reference.`
	msgBadEncoding      = `There was an issue with the content of the DOI file (datacite.yml). This might mean that the encoding is wrong. Please see <a href="https://gin.g-node.org/G-Node/Info/wiki/DOIfile">the DOI guide</a> for detailed instructions or contact gin@g-node.org for assistance.`

	msgSubmitError     = "An internal error occurred while we were processing your request.  The G-Node team has been notified of the problem and will attempt to repair it and process your request.  We may contact you for further information regarding your request.  Feel free to <a href=mailto:gin@g-node.org>contact us</a> if you would like to provide more information or ask about the status of your request."
	msgSubmitFailed    = "An internal error occurred while we were processing your request.  Your request was not submitted and the service failed to notify the G-Node team.  Please <a href=mailto:gin@g-node.org>contact us</a> to report this error."
	msgNoTemplateError = "An internal error occurred while we were processing your request.  The G-Node team has been notified of the problem and will attempt to repair it and process your request.  We may contact you for further information regarding your request.  Feel free to contact us at gin@g-node.org if you would like to provide more information or ask about the status of your request."
	// Log Prefixes
	lpAuth    = "GinOAP"
	lpStorage = "Storage"
	lpMakeXML = "MakeXML"
)
