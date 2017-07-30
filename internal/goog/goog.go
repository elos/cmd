package goog

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"

	calendar "google.golang.org/api/calendar/v3"
	gmail "google.golang.org/api/gmail/v1"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const elosClientFile = `{"installed":{"client_id":"232544408124-m2utq9itie2iv0809vvkk0ra9v6vut27.apps.googleusercontent.com","project_id":"eloscloud","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://accounts.google.com/o/oauth2/token","auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_secret":"iZbE9VacfmbvzDhhsnkmpmPh","redirect_uris":["urn:ietf:wg:oauth:2.0:oob","http://localhost"]}}`

var (
	GmailTokenPath    string
	CalendarTokenPath string
)

func init() {
	u, err := user.Current()
	if err != nil {
		log.Fatalf("user.Current error: %v", err)
	}
	googDir := filepath.Join(u.HomeDir, "elos", "goog")
	os.MkdirAll(googDir, 0700)
	GmailTokenPath = filepath.Join(googDir, "gmail-token")
	CalendarTokenPath = filepath.Join(googDir, "calendar-token")
}

// GetTokenFromWeb prompts the user to go through the Google
// Oauth flow in their web browser.
func GetTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %v", err)
	}

	return tok, nil
}

func Gmail(ctx context.Context) (*gmail.Service, error) {
	conf, err := google.ConfigFromJSON([]byte(elosClientFile), gmail.MailGoogleComScope)
	if err != nil {
		return nil, err
	}
	t, err := token(conf, GmailTokenPath)
	if err != nil {
		return nil, err
	}
	return gmail.New(conf.Client(ctx, t))
}

func Calendar(ctx context.Context) (*calendar.Service, error) {
	conf, err := google.ConfigFromJSON([]byte(elosClientFile), calendar.CalendarScope)
	if err != nil {
		return nil, err
	}
	t, err := token(conf, CalendarTokenPath)
	if err != nil {
		return nil, err
	}
	return calendar.New(conf.Client(ctx, t))
}

func newToken(conf *oauth2.Config, f string) (*oauth2.Token, error) {
	t, err := GetTokenFromWeb(conf)
	if err != nil {
		return nil, err
	}
	bs, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(f, bs, 0600); err != nil {
		return nil, err
	}
	return t, nil
}

func token(conf *oauth2.Config, p string) (*oauth2.Token, error) {
	t := new(oauth2.Token)
	f, err := os.Open(p)
	if err != nil {
		return newToken(conf, p)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(t); err != nil {
		return newToken(conf, p)
	}
	return t, nil
}
