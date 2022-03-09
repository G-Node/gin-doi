package gdtmpl

// ChecklistFile is the template for rendering the semi-automated
// DOI registration process checklist.
const ChecklistFile = `# Part 1 - pre registration

## Base request information
-[ ] check if the following information is correct; re-run script otherwise with updated config

    DOI request
    - Repository: {{ .CL.Repoown }}/{{ .CL.Repo }}
    - User: ({{ .CL.Userfullname }})
    - Email address: {{ .CL.Email }}
    - DOI XML: {{ .CL.Doiserver }}:/data/doi/10.12751/g-node.{{ .CL.Regid }}/doi.xml
    - DOI target URL: https://doi.gin.g-node.org/10.12751/g-node.{{ .CL.Regid }}

    - Request Date (as in doi.xml): {{ .CL.Regdate }}

## Base pre-registration checks

- check the datacite content at 
  https://gin.g-node.org/{{ .CL.Repoown }}/{{ .CL.Repo }}
    -[ ] repo is eligible to be published via GIN DOI
    -[ ] the repo name is sufficiently unique to avoid clashes when 
         forking to the DOI GIN user.
    -[ ] resourceType e.g. Dataset fits the repository
    -[ ] license in datacite.yml and LICENSE file match
    -[ ] author list includes requester, otherwise ask requester for confirmation
    -[ ] ORCIDs look reasonable / are valid
    -[ ] title is useful and has no typos
    -[ ] keywords have no typos
    -[ ] automated issues are all addressed

- on the GIN server check annex content
    -[ ] /gindata/annexcheck /gindata/gin-repositories/{{ .RepoownLower }}/{{ .RepoLower }}.git

- on the DOI server ({{ .CL.Doiserver }}) make sure all information has been properly downloaded 
  to the staging directory and all annex files are unlocked and the content is present:
    -[ ] {{ .CL.Dirdoiprep }}/annexcheck {{ .SemiDOIDirpath }}
    -[ ] if locked content has been found and unlocked by the doi-server, {{ .SemiDOICleanup }}
         will contain a second folder named "{{ .RepoLower }}_unlocked". The zip file was created
         with the content of this repository which should contain only unlocked files.
         DO NOT USE the "unlocked" folder to upload to the DOI fork later on.
    - identify "normal" git annex issues e.g. locked or missing annex content
    -[ ] cd {{ .SemiDOICleanup }}/{{ .RepoLower }}
    -[ ] gin git annex find --locked
    -[ ] gin git annex find --not --in=here
    -[ ] find {{ .SemiDOICleanup }} -type l -print
    -[ ] find {{ .SemiDOICleanup }} -type f -size -100c -print0 | xargs -0 grep -i annex.objects
    -[ ] grep annex.objects $(find {{ .SemiDOICleanup }} -type f -size -100c -print)
    -[ ] check that the content size of the repository and the created zip file matches
    -[ ] if there still are symlinks present or the content size does not match up, the zip
         file does not contain all required data. Run the next steps - the script will
         download all missing information and upload to the DOI fork. When recreating the
         zip file, all files will be manually unlocked first.
    - approximate the required zip size via the git annex file size and the repository size
    -[ ] gin git annex info --fast .
    -[ ] du -chL --exclude=.git* .
    -[ ] ls -lahrt  {{ .CL.Dirdoi }}/10.12751/g-node.{{ .CL.Regid }}/
    - check the DOI directory content
      -[ ] zip file created in {{ .CL.Dirdoi }}/10.12751/g-node.{{ .CL.Regid }}
      -[ ] check zip file content
           unzip -vl {{ .CL.Dirdoi }}/10.12751/g-node.{{ .CL.Regid }}/10.12751_g-node.{{ .CL.Regid }}.zip
      -[ ] note zip size
    - check potential dataset zip files for issues
      find {{ .SemiDOICleanup }} -name "*.zip" -ls -exec unzip -P "" -t {} \; > $HOME/logs/zipcheck_{{ .CL.Regid }}.log
      echo "Valid zips: $(cat $HOME/logs/zipcheck_{{ .CL.Regid }}.log | grep "No errors detected" | wc -l)/$(find . -name "*.zip" | wc -l)"
    - if the number of valid zips does not match the number of total zips, check the logfile for details

## Semi-automated DOI or DOI update
- use this section if there are no technical or other issues with the DOI request 
  and skip the 'Full DOI' section.
- also use this section if there were no issues and an update to an existing DOI has
  been requested. The 'doiforkupload' script does both initial upload and update.

-[ ] remove {{ .CL.Dirdoi }}/10.12751/g-node.{{ .CL.Regid }}/.htaccess

- access https://doi.gin.g-node.org/10.12751/g-node.{{ .CL.Regid }}
    -[ ] check landing page in general
    -[ ] check title, license name
    -[ ] check all links that should work at this stage
    -[ ] check zip download and compare size on server with size in 'doi.xml'

-[ ] manually fork repository to the 'doi' gin user
    - log on to gin.g-node.org using the "doi" user
    - fork https://gin.g-node.org/{{ .CL.Repoown }}/{{ .CL.Repo }}

-[ ] log on to the DOI server ({{ .CL.Doiserver }}) and move to {{ .CL.Dirdoiprep }}
-[ ] fetch git and annex content and upload annex content to the DOI fork repo.
     use screen to avoid large down- and uploads to be interrupted.
     use CTRL+a+d to switch out of screen sessions without interruption.
     use either the logfile or 'htop' to check on the status of the download/upload.
    - screen -S {{ .SemiDOIScreenID }}
    - sudo su root
    - ./doiforkupload {{ .SemiDOIDirpath }} > {{ .Forklog }}
-[ ] after detaching from the session, check the log file until the upload starts to avoid
     any security check issues.
     Also read the commit hash comparison line to check if the content of the repo has
     been changed after the DOI request has been submitted.
     tail -f {{ .Forklog }}
-[ ] if a) the logfile contains the line "repo was not at the DOI request state" the
     repository was changed after the DOI request and the uploaded archive content will
     most likely differ from the zip file content. If b) the 'tree' command showed symlinks or 
     missing content, the zip file will also not contain the file content for all files.
       In this case use the 'makezip' bash script to recreate the zip file and 
     copy it to the the DOI hosting folder.
-[ ] once the upload is done, check that the git tag has been created on the DOI fork repository at
     https://gin.g-node.org/doi/{{ .CL.Repo }}.

- cleanup directory once tagging is done
    -[ ] sudo rm {{ .SemiDOICleanup }} -r
    -[ ] sudo mv {{ .CL.Dirdoiprep }}/{{ .Logfiles }} $HOME/logs/
    -[ ] cleanup screen session: screen -XS {{ .SemiDOIScreenID }} quit

-[ ] Check link to archive repo on the DOI landing page works:
    https://doi.gin.g-node.org/10.12751/g-node.{{ .CL.Regid }}

-[ ] issue comment on https://gin.g-node.org/G-Node/DOImetadata/issues
     New publication request: {{ .RepoownLower }}/{{ .RepoLower }} (10.12751/g-node.{{ .CL.Regid }})

     This repository is prepared for the DOI registration.

## Full DOI
- This usually has to be done when
  a) the semi-automated process has failed or
  b) the user requested changes but needs to keep the originally issued DOI

-[ ] manually fork repository to the 'doi' gin user
    - log on to gin.g-node.org using the "doi" user
    - fork https://gin.g-node.org/{{ .CL.Repoown }}/{{ .CL.Repo }}

-[ ] log on to the DOI server ({{ .CL.Doiserver }}) and move to {{ .CL.Dirdoiprep }}
-[ ] fetch git and annex content and upload annex content to the DOI fork repo.
     use screen to avoid large down- and uploads to be interrupted.
     use CTRL+a+d to switch out of screen sessions without interruption.
     use either the logfile or 'htop' to check on the status of the download/upload.
    - screen -S {{ .FullDOIScreenID }}
    - sudo su root
    - ./syncannex {{ .CL.Repoown }}/{{ .CL.Repo }} > {{ .Forklog }}

-[ ] check downloaded data; if any of the checks fail, the DOI fork has to be deleted and the 
     process repeated after the issue has been addressed
    -[ ] {{ .CL.Dirdoiprep }}/annexcheck {{ .CL.Dirdoiprep }}/{{ .CL.Repo }}
    - identify "normal" git annex issues e.g. locked or missing annex content
    -[ ] cd {{ .CL.Dirdoiprep }}/{{ .CL.Repo }}
    -[ ] gin git annex find --locked
    -[ ] gin git annex find --not --in=here
    -[ ] find {{ .CL.Dirdoiprep }}/{{ .CL.Repo }} -type l -print
    -[ ] find {{ .CL.Dirdoiprep }}/{{ .CL.Repo }} -type f -size -100c -print0 | xargs -0 grep -i annex.objects
    -[ ] grep annex.objects $(find {{ .CL.Dirdoiprep }}/{{ .CL.Repo }} -type f -size -100c -print)
    - check potential dataset zip files for issues
      find {{ .CL.Dirdoiprep }}/{{ .CL.Repo }} -name "*.zip" -ls -exec unzip -P "" -t {} \; > $HOME/logs/zipcheck_{{ .CL.Regid }}.log
      echo "Valid zips: $(cat $HOME/logs/zipcheck_{{ .CL.Regid }}.log | grep "No errors detected" | wc -l)/$(find . -name "*.zip" | wc -l)"
    - if the number of valid zips does not match the number of total zips, check the logfile for details

-[ ] create DOI zip file
    - screen -r {{ .FullDOIScreenID }}
    - sudo ./makezip {{ .RepoLower }} > {{ .Ziplog }}

- approximate the required zip size via the git annex file size and the repository size and compare to the created zip size
    -[ ] cd {{ .CL.Dirdoiprep }}/{{ .CL.Repo }}
    -[ ] gin git annex info --fast .
    -[ ] du -chL --exclude=.git* .
    -[ ] ls -lahrt {{ .CL.Dirdoiprep }}/{{ .CL.Repo }}/*.zip

-[ ] make sure there is no zip file in the target directory left 
     from the previous registration process.

-[ ] sudo mv {{ .RepoLower }}.zip {{ .Zipfile }}

- create release tag on the DOI repository; run all commands using 'gin git ...' 
  to avoid issues with local git annex or other logged git users.
    -[ ] cd {{ .CL.Dirdoiprep }}/{{ .RepoLower }}
    -[ ] check that "doi" is the set origin: sudo gin git remote -v
    -[ ] sudo gin git tag 10.12751/g-node.{{ .CL.Regid }}
    -[ ] sudo gin git push --tags origin

- cleanup directory once tagging is done
    -[ ] sudo rm {{ .FullDOIDirpath }} -r
    -[ ] sudo mv {{ .CL.Dirdoiprep }}/{{ .Logfiles }} $HOME/logs/
    -[ ] cleanup screen session: screen -XS {{ .FullDOIScreenID }} quit

-[ ] edit {{ .CL.Dirdoi }}/10.12751/g-node.{{ .CL.Regid }}/doi.xml file to reflect
     any changes in the repo datacite.yml file.
    - include the actual size of the zip file
    - check proper title and proper license
    - any added or updated funding or reference information
    - any changes to the 'resourceType'

- remove the .htaccess file
- re-create the DOI landing page in the server staging directory
    -[ ] cd $HOME/staging
    -[ ] sudo {{ .CL.Dirdoiprep }}/gindoid make-html https://doi.gin.g-node.org/10.12751/g-node.{{ .CL.Regid }}/doi.xml
    -[ ] sudo mv index.html {{ .CL.Dirdoi }}/10.12751/g-node.{{ .CL.Regid }}/index.html

- https://doi.gin.g-node.org/10.12751/g-node.{{ .CL.Regid }}
    -[ ] check page access, size, title, license name
    -[ ] check all links that should work at this stage
    -[ ] check zip download and suggested size

-[ ] issue comment on https://gin.g-node.org/G-Node/DOImetadata/issues
     New publication request: {{ .RepoownLower }}/{{ .RepoLower }} (10.12751/g-node.{{ .CL.Regid }})

     This repository is prepared for the DOI registration.

# Part 2 - post registration
-[ ] connect to DOI server ({{ .CL.Doiserver }})
- update the G-Node/DOImetadata repository
  cd {{ .CL.Dirdoiprep }}/DOImetadata
  sudo gin download
- update site listing page, google sitemap and keywords
  -[ ] create a clean server staging directory
    sudo mkdir -p $HOME/staging/g-node.{{ .CL.Regid }}
    cd $HOME/staging/g-node.{{ .CL.Regid }}
  -[ ] create all required files
    -[ ] sudo {{ .CL.Dirdoiprep }}/gindoid make-all {{ .CL.Dirdoiprep }}/DOImetadata/*.xml
    -[ ] check index.html and urls.txt file
    -[ ] sudo mv {{ .CL.Dirdoi }}/keywords {{ .CL.Dirdoi }}/keywords_
    -[ ] sudo mv $HOME/staging/g-node.{{ .CL.Regid }}/keywords/ {{ .CL.Dirdoi }}
    -[ ] sudo mv $HOME/staging/g-node.{{ .CL.Regid }}/index.html {{ .CL.Dirdoi }}
    -[ ] sudo mv $HOME/staging/g-node.{{ .CL.Regid }}/urls.txt {{ .CL.Dirdoi }}
  -[ ] check landing page and keywords online: https://doi.gin.g-node.org
  -[ ] cleanup previous keywords
    sudo rm {{ .CL.Dirdoi }}/keywords_ -r
  -[ ] cleanup the staging directory
    cd $HOME/staging
    sudo rm g-node.{{ .CL.Regid }} -r

-[ ] git commit all changes in {{ .CL.Dirdoi }}
    - sudo git add 10.12751/g-node.{{ .CL.Regid }}/
    - sudo git commit -m "New dataset: 10.12751/g-node.{{ .CL.Regid }}"

-[ ] commit keyword and index page changes
    - sudo git add keywords/
    - sudo git add index.html
    - sudo git add urls.txt
    - sudo git commit -m "Update index and keyword pages"

-[ ] set zip to immutable
    sudo chattr +i {{ .CL.Dirdoi }}/10.12751/g-node.{{ .CL.Regid }}/10.12751_g-node.{{ .CL.Regid }}.zip

-[ ] cleanup any leftover directories from previous versions 
     of this dataset in the {{ .CL.Dirdoi }}/10.12751/ and 
    {{ .CL.Dirdoiprep }}/10.12751/ directories.

-[ ] email to user (check below)

-[ ] close all related issues on https://gin.g-node.org/G-Node/DOImetadata/issues
     New publication request: {{ .RepoownLower }}/{{ .RepoLower }} (10.12751/g-node.{{ .CL.Regid }})

     Publication finished and user informed.

# Part 3 - eMail to user
-[ ] make sure the publication reference text does apply, remove otherwise

{{ .CL.Email }}

CC: gin@g-node.org

Subject: DOI registration complete - {{ .CL.Repoown }}/{{ .CL.Repo }}

Dear {{ .CL.Userfullname }},

Your dataset with title
  {{ .CL.Title }}

has been successfully registered.

The DOI for the dataset is
  https://doi.org/10.12751/g-node.{{ .CL.Regid }}

Please always reference the dataset by its DOI (not the link to the repository) and cite the dataset as
  {{ .CL.Citation }} ({{ .Citeyear }}) {{ .CL.Title }}. G-Node. https://doi.org/10.12751/g-node.{{ .CL.Regid }}

If this is data supplementing a publication and if you haven't done so already, we kindly request that you:
- include the new DOI of this dataset in the publication as a reference, and
- update the datacite file of the registered dataset to reference the publication, including its DOI, once it is known.

The latter will result in a link in the Datacite database to your publication and will increase its discoverability.

Best regards,

  German Neuroinformatics Node
`
