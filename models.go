package main

import (
	"html/template"
)

//Config stores Host/Port Information, the location of the user DB and settings for the cookiestores
//and is loaded from a config file using loadConfiguration
//Host and Port are used for parsing and delivering pages, Key/Secret pairs are obtained from the provider when registering the application.
type Config struct {
	Host          string `json:"host"`
	Port          string `json:"port"`
	GitHubKey     string `json:"gitHubKey"`
	GitHubSecret  string `json:"githHubSecret"`
	GitLabKey     string `json:"gitLabKey"`
	GitLabSecret  string `json:"gitLabSecret"`
	GitLabScope   string `json:"gitLabScope"` //for accessing GitLab user information this has to be "read_user"
	MaxAge        int    `json:"maxAge"`      //sets the lifetime of the brucheion session
	UserDB        string `json:"userDB"`
	//	GoogleKey	    string `json:"googleKey"`
	//	GoogleSecret  string `json:"googleSecret"`
	OrthographyNormalisationFilenames string `json:"orthographyNormalisationFilenames"`
}

//Transcription is the container for a transcription and context metadata
type Transcription struct {
	CTSURN        string
	Transcriber   string
	Transcription string
	Previous      string
	Next          string
	First         string
	Last          string
	ImageRef      []string
	TextRef       []string
	ImageJS       string
	CatID         string
	CatCit        string
	CatGroup      string
	CatWork       string
	CatVers       string
	CatExmpl      string
	CatOn         string
	CatLan        string
}

// CompPage stores Information necessary to parse and display a compare page
// used in comparePage and consolidatePage
// and corresponding pageloaders
type CompPage struct {
	User      string
	Title     string
	Text      template.HTML
	Host      string
	CatID     string
	CatCit    string
	CatGroup  string
	CatWork   string
	CatVers   string
	CatExmpl  string
	CatOn     string
	CatLan    string
	User2     string
	Title2    string
	Text2     template.HTML
	CatID2    string
	CatCit2   string
	CatGroup2 string
	CatWork2  string
	CatVers2  string
	CatExmpl2 string
	CatOn2    string
	CatLan2   string
}

// Page stores Information necessary to parse and display a general purpose page
// used in CrudPage, Edit2Page, EditCatPage,  EditPage, Multipage, TreePage, ViewPage
// and corresponding pageloaders
type Page struct {
	User         string
	Title        string
	ImageJS      string
	ImageScript  template.HTML
	ImageHTML    template.HTML
	TextHTML     template.HTML
	InTextHTML   template.HTML
	Text         template.HTML
	Previous     string
	Next         string
	PreviousLink template.HTML
	NextLink     template.HTML
	First        string
	Last         string
	Host         string
	ImageRef     string
	CatID        string
	CatCit       string
	CatGroup     string
	CatWork      string
	CatVers      string
	CatExmpl     string
	CatOn        string
	CatLan       string
}

// LoginPage stores Information necessary to parse and display a login page
// Used in loginGET, loginPOST, authCallback
type LoginPage struct {
	BUserName    string //The username that the user chooses to work with within Brucheion
	Provider     string //The login provider
	HrefUserName string //Combination {user}_{provider} as displayed in link
	Message      string //Message to be displayed according to login scenario
	Host         string //Port of the Link
	Title        string //Title of the website
	NoAuth       bool   //representation of the noAuth flag
}

//BrucheionUser stores Information about the logged in Brucheion user
type BrucheionUser struct {
	BUserName      string //The username choosen by user to use Brucheion with
	Provider       string //The provider used for authentification
	PUserName      string //The username used for login with the provider
	ProviderUserID string //The UserID issued by Provider
}

//Validation stores the result of a user or username validation
type Validation struct {
	Message       string //Message according to outcome of validation
	ErrorCode     bool   //Was an error encountered during validation (something did not match)?
	BUserInUse    bool   //func validateUser: Is the BrucheionUser to be found in the DB?
	SameProvider  bool   //func validateUser: Is the chosen provider the same as the providersaved in DB?
	PUserInUse    bool   //func validateUser: Is the ProviderUser to be found in the DB?
	BPAssociation bool   //func validateNoAuthUser: Is the choosen NoAuthUser already in use with a provider login?
}

// multi alignment testing

// Alignments is a named container for Aligment structs
//Used in MultiPage and nwa2
type Alignments struct {
	Alignment []Alignment
	Name      []string
}

//Alignment is a container for alignment results
type Alignment struct {
	Source []string
	Target []string
	Score  []float32
}

// BoltCatalog contains all metadata of a CITE URN and is
//used in LoadCEX and page functions
type BoltCatalog struct {
	URN           string `json:"urn"`
	Citation      string `json:"citationScheme"`
	GroupName     string `json:"groupName"`
	WorkTitle     string `json:"workTitle"`
	VersionLabel  string `json:"versionLabel"`
	ExemplarLabel string `json:"exemplarLabel"`
	Online        string `json:"online"`
	Language      string `json:"language"`
}
