[![Docker Automated buil](https://img.shields.io/docker/automated/cgars/gin-doi.svg)](https://hub.docker.com/r/cgars/gin-doi/builds/)

# gin-doi
G-Node DOI Service

## What is a Doi and why should i use it
> A Digital Object Identifier or DOI is a persistent identifier or handle used to uniquely identify objects, standardized by the ISO. An implementation of the Handle System,[2][3] DOIs are in wide use mainly to identify academic, professional, and government information, such as journal articles, research reports and data sets, and official publications though they also have been used to identify other types of information resources, such as commercial videos.

## What is gin-doi
gin-doi is the G-Node Infrastructure doi service. A Service which can copy your public repository, packs everything into anarchive file, stores it in a super save location and provides you with a doi such that you can cite this data. 
gin-doi fulfills the [Data Cite](https://www.datacite.org/) standard which (according to Wikipedia) tries to:
* Establish easier access to research data on the Internet
* Increase acceptance of research data as legitimate, citable contributions to the scholarly record
* Support data archiving that will permit results to be verified and re-purposed for future study.

## What is needed
To get a doi you need to provide a file called .cloudberry which needs to be put (and pushed) into the root of your repository.
This file needsa to be a valid [YAML](https://en.wikipedia.org/wiki/YAML) file and should look  [like this one](https://github.com/cgars/gin-doi/blob/master/tmpl/example_cloudberry.yml).
You need to provide  at least the following entries:
* authors
* title
* description
* keywords
* license
* references

**please note that the keys (authors, title, description, etc.) need to be lower case and followed by a colon.**

Furthermore the repository you want to get a DOI for must be public.

You can find datasets that are already doified [here](http://doid.gin.g-node.org). Join us, make data great (again?)! 
