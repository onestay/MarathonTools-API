package social

import "github.com/onestay/MarathonTools-API/api/common"

// Controller holds all the info and methods
type Controller struct {
	twitchInfo *twitchInfo
	base       *common.Controller
}

type twitchInfo struct {
	ClientID     string
	ClientSecret string
	Scope        string
	RedirectURI  string
}

// NewSocialController will return a new social controller
func NewSocialController(twitchClientID, twitchClientSecret string, b *common.Controller) *Controller {
	t := &twitchInfo{
		ClientID:     twitchClientID,
		ClientSecret: twitchClientSecret,
		Scope:        "channel_editor",
		RedirectURI:  "http://localhost:4000/#/dashboard/config/social/twitch",
	}

	return &Controller{
		twitchInfo: t,
		base:       b,
	}
}
