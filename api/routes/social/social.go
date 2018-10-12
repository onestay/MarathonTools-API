package social

import (
	"fmt"

	"github.com/onestay/MarathonTools-API/api/common"

	"github.com/dghubble/oauth1"
	"github.com/dghubble/oauth1/twitter"
)

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
func NewSocialController(twitchClientID, twitchClientSecret, twitchCallback, twitterKey, twitterSecret, twitterCallback string, b *common.Controller) *Controller {
	t := &twitchInfo{
		ClientID:     twitchClientID,
		ClientSecret: twitchClientSecret,
		Scope:        "channel_editor channel_read channel_commercial",
		RedirectURI:  twitchCallback,
	}

	tw := &oauth1.Config{
		ConsumerKey:    twitterKey,
		ConsumerSecret: twitterSecret,
		CallbackURL:    twitterCallback,
		Endpoint:       twitter.AuthorizeEndpoint,
	}

	c := Controller{
		twitchInfo:  t,
		base:        b,
		twitterInfo: tw,
	}

	go c.comReciever()

	return &c
}

func (sc Controller) comReciever() {
	for {
		i := <-sc.base.SocialUpdatesChan
		if i == 1 {
			err := sc.twitchUpdateInfo()
			if err != nil {
				sc.base.LogError("while updating twitch info", err, true)
			}
		} else if i == 2 {
			err := sc.twitterSendUpdate()
			if err != nil {
				sc.base.LogError("while sending tweet update", err, true)
			}
		} else if i == 0 {
			fmt.Println(i)
		}
	}
}
