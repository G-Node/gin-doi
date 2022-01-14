package main

// Default configuration struct containing non problematic test values
type checklist struct {
	// Entries required for every DOI request
	// Paste basic information from the corresponding issue on
	//   https://gin.g-node.org/G-Node/DOIMetadata
	// Automated registration [id] from "10.12751/g-node.[id]"
	Regid string `yaml:"reg_id"`
	// Repository owner
	Repoown string `yaml:"repo_own"`
	// Repository name
	Repo string `yaml:"repo"`
	// Date issued from doi.xml; Format YYYY-MM-DD
	Regdate string `yaml:"reg_date"`
	// DOI requestee email address
	Email string `yaml:"email"`
	// DOI requestee full name
	Userfullname string `yaml:"user_full_name"`
	// Entries that are usually handled automatically via repo datacite entry
	// DOI request title; usually handled automatically via repo datacite entry
	Title string `yaml:"title"`
	// Author citation list; usually handled automatically via repo datacite entry
	Citation string `yaml:"citation"`
	// Entries that are set once and remain unchanged for future DOI requests
	// User working on the DOI server
	Serveruser string `yaml:"server_user"`
	// Local staging dir to create index and keyword pages
	Dirlocalstage string `yaml:"dir_local_stage"`
	// Full ssh access name of the server hosting the GIN server instance
	Ginserver string `yaml:"gin_server"`
	// Full ssh access name of the server hosting the DOI server instance
	Doiserver string `yaml:"doi_server"`
	// DOI Server repo preparation directory
	Dirdoiprep string `yaml:"dir_doi_prep"`
	// DOI Server root doi hosting directory
	Dirdoi string `yaml:"dir_doi"`
}
