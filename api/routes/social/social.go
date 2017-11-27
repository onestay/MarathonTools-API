package social

import "github.com/onestay/MarathonTools-API/api/common"

import "github.com/dghubble/oauth1"
import "github.com/dghubble/oauth1/twitter"

// Controller holds all the info and methods
type Controller struct {
	twitchInfo  *twitchInfo
	twitterInfo *oauth1.Config
	base        *common.Controller
}

type twitchInfo struct {
	ClientID     string
	ClientSecret string
	Scope        string
	RedirectURI  string
}

type twitterInfo struct {
	ConsumerKey    string
	ConsumerSecret string
	CallbackURL    string
	Endpoint       oauth1.Endpoint
}

// NewSocialController will return a new social controller
func NewSocialController(twitchClientID, twitchClientSecret string, b *common.Controller, twitterKey, twitterSecret string) *Controller {
	t := &twitchInfo{
		ClientID:     twitchClientID,
		ClientSecret: twitchClientSecret,
		Scope:        "channel_editor",
		RedirectURI:  "http://localhost:4000/dashboard/config/social/twitch",
	}

	tw := &oauth1.Config{
		ConsumerKey:    twitterKey,
		ConsumerSecret: twitterSecret,
		CallbackURL:    "http://localhost:4000/dashboard/config/social/twitter",
		Endpoint:       twitter.AuthorizeEndpoint,
	}

	return &Controller{
		twitchInfo:  t,
		base:        b,
		twitterInfo: tw,
	}
}
