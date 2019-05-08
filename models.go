package main

import (
	"html/template"

	"github.com/ThomasK81/gocite"
)

//All necessary structs may be collected in this file for better overview.
//It might be usefull to group them in a senseful way.

// *** Login ***

//Config stores Host/Port Information, the location of the user DB and settings for the cookiestores
//and is loaded from a config file using loadConfiguration
//Host and Port are used for parsing and delivering pages, Key/Secret pairs are obtained from the provider when registering the application.
type Config struct {
	Host                              string            `json:"host"`
	Port                              string            `json:"port"`
	GitHubKey                         string            `json:"gitHubKey"`
	GitHubSecret                      string            `json:"githHubSecret"`
	GitLabKey                         string            `json:"gitLabKey"`
	GitLabSecret                      string            `json:"gitLabSecret"`
	GitLabScope                       string            `json:"gitLabScope"` //for accessing GitLab user information this has to be "read_user"
	MaxAge                            int               `json:"maxAge"`      //sets the lifetime of the brucheion session
	UserDB                            string            `json:"userDB"`
	OrthographyNormalisationFilenames map[string]string `json:"orthographyNormalisationFilenames"`
	UseNormalization                  bool              `json:"useNormalization"`
	//	GoogleKey	    string `json:"googleKey"`
	//	GoogleSecret  string `json:"googleSecret"`
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

// *** Transcription Desk ***

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

//imageCollection is the container for image collections along with their URN and name as strings
type imageCollection struct {
	URN        string  `json:"urn"`
	Name       string  `json:"name"`
	Collection []image `json:"images"`
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

//*** Pages ****

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

// *** Multi Alignment Testing

//Alignment is a container for alignment results
type Alignment struct {
	Source []string
	Target []string
	Score  []float32
}

// Alignments is a named container for Aligment structs
//Used in MultiPage and nwa2
type Alignments struct {
	Alignment []Alignment
	Name      []string
}

// *** Treebank containers ***

type TreeNode struct {
	Identifier string     `json:"name"`
	SentenceID string     `json:"sentence"`
	CTSID      string     `json:"CTSID"`
	WordToken  WordToken  `json:"token"`
	Lemma      Lemma      `json:"lemma"`
	Children   []TreeNode `json:"children"`
	Parent     []TreeNode `json:"parent"`
}

type WordToken struct {
	Identifier string `json:"treeTokenID"`
	Text       string `json:"treeToken"`
	Relation   string `json:"relation"`
	POStag     string `json:"postag"`
}

type Lemma struct {
	Identifier string `json:"treeLemmaID"`
	Text       string `json:"treeLemma"`
}

type Sentence struct {
	Identifier    string
	CTSIdentifier string
	Text          string
}

// *** Work Container ***

//cexMeta is the container for CEX metadata (work metadata). Used for saving new URNs with newWork
//or changing metatdata with updateWorkMeta
type cexMeta struct {
	URN, CitationScheme, GroupName, WorkTitle, VersionLabel, ExemplarLabel, Online, Language string
}

// *** CITE Data Containers ***
//These are probably going to be retired altogether.

//BoltData is the container for CITE data imported from CEX files and is used in LoadCEX
type BoltData struct {
	Bucket  []string // workurn
	Data    []gocite.Work
	Catalog []BoltCatalog
}

//BoltWork is the container for BultURNs and their associated keys and is used in LoadCEX
type BoltWork struct {
	Key  []string // cts-node urn
	Data []BoltURN
}

//BoltURN is the container for a textpassage along with its URN, its image reference,
//and some information on preceding and anteceding works.
//Used for loading and saving CEX files, for pages, and for nodes
type BoltURN struct {
	URN      string   `json:"urn"`
	Text     string   `json:"text"`
	LineText []string `json:"linetext"`
	Previous string   `json:"previous"`
	Next     string   `json:"next"`
	First    string   `json:"first"`
	Last     string   `json:"last"`
	Index    int      `json:"sequence"`
	ImageRef []string `json:"imageref"`
}

//BoltJSON is a string representation of a JSON used in BoltRetrieve
type BoltJSON struct {
	JSON string
}
