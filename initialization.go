package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"

	"github.com/boltdb/bolt"
)

type Config struct {
	Host                              string            `json:"host"`
	Port                              string            `json:"port"`
	MaxAge                            int               `json:"maxAge"`
	UserDB                            string            `json:"userDB"`
	OrthographyNormalisationFilenames map[string]string `json:"orthographyNormalisationFilenames"`
	UseNormalization                  bool              `json:"useNormalization"`
}

type ProviderAccess struct {
	Key    string `json:"key"`
	Secret string `json:"secret"`
}
type Providers struct {
	GitHub ProviderAccess `json:"github"`
	GitLab ProviderAccess `json:"gitlab"`
	Google ProviderAccess `json:"google"`
}

//go:embed providers.json
var provdata []byte

// loadProviders loads provider keys and secrets from the embedded providers JSON data and returns a decoded Providers
// structure.
func loadProviders() (Providers, error) {
	var p Providers
	r := bytes.NewReader(provdata)
	d := json.NewDecoder(r)
	err := d.Decode(&p)
	return p, err
}

// loadConfiguration loads and parses the JSON configuration file and returns a Config structure.
func loadConfiguration(file string) (Config, error) {
	var c Config

	cf, err := os.Open(file)
	defer cf.Close()
	if err != nil {
		return Config{}, err
	}

	jsonParser := json.NewDecoder(cf)
	err = jsonParser.Decode(&c)
	return c, err
}

func getSessionKey() []byte {
	keyFile := filepath.Join(dataPath, ".session-key")
	content, err := ioutil.ReadFile(keyFile)
	if err != nil {
		key := securecookie.GenerateRandomKey(64)
		_ = ioutil.WriteFile(keyFile, key, 0666)
		return key
	}
	return content
}

// getCookieStore sets up and returns a cookiestore. The maxAge is defined by what was defined in config.json.
//Todo: Errorhandling
func getCookieStore(maxAge int) sessions.Store {
	key := getSessionKey()
	cookieStore := sessions.NewCookieStore([]byte(key)) //Get CookieStore from sessions package
	cookieStore.Options.HttpOnly = true                 //Ensures that Cookie can not be accessed by scripts
	cookieStore.MaxAge(maxAge)                          //Sets the maxAge of the session/cookie

	return cookieStore
}

//initializeUserDB should be called once during login attempt to make sure that all buckets are in place.
func initializeUsersDB() error {
	log.Println("Initializing UserDB")
	db, err := openBoltDB(config.UserDB)
	if err != nil {
		return err
	}

	//create the three buckets needed: users, GitHub, GitLab
	db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("users")) //create a new bucket to store new user
		if err != nil {
			return fmt.Errorf("Failed creating bucket users: %s", err)
		}
		_ = bucket //to have done something with the bucket (avoiding 'username declared and not used' error and ineffasign notice)
		bucket, err = tx.CreateBucketIfNotExists([]byte("GitHub"))
		if err != nil {
			return fmt.Errorf("Failed creating bucket GitHub: %s", err)
		}
		_ = bucket //to have done something with the bucket (avoiding 'username declared and not used' error and ineffasign notice)
		bucket, err = tx.CreateBucketIfNotExists([]byte("GitLab"))
		if err != nil {
			return fmt.Errorf("Failed creating bucket GitLab: %s", err)
		}
		_ = bucket //to have done something with the bucket (avoiding 'username declared and not used' error and ineffasign notice)
		bucket, err = tx.CreateBucketIfNotExists([]byte("Google"))
		if err != nil {
			return fmt.Errorf("Failed creating bucket Google: %s", err)
		}
		_ = bucket //to have done something with the bucket (avoiding 'username declared and not used' error and ineffasign notice)

		return nil //if all went well, error can be returned with <nil>
	})
	db.Close() //always remember to close the db
	//fmt.Println("DB closed")
	return nil
}

//initializeSession will create and return the session and set the session options
func initializeSession(req *http.Request) (*sessions.Session, error) {
	log.Println("Initializing session: " + SessionName)
	session, err := BrucheionStore.Get(req, SessionName)
	if err != nil {
		fmt.Printf("InitializeSession: Error getting the session: %s\n", err)
		return nil, err
	}
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   config.MaxAge,
		HttpOnly: true}
	return session, nil
}
