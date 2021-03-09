package social

import (
	"fmt"

	"github.com/julienschmidt/httprouter"
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

func (sc Controller) registerRoutes(r *httprouter.Router)  {
	r.GET("/social/twitch/oauthurl", sc.TwitchOAuthURL)
	r.GET("/social/twitch/verify", sc.TwitchCheckForAuth)
	r.POST("/social/twitch/auth", sc.TwitchGetToken)
	r.DELETE("/social/twitch/token", sc.TwitchDeleteToken)
	r.GET("/social/twitch/executetemplate", sc.TwitchExecuteTemplate)
	r.PUT("/social/twitch/update", sc.TwitchUpdateInfo)
	r.PUT("/social/twitch/settings", sc.TwitchSetSettings)
	r.GET("/social/twitch/settings", sc.TwitchGetSettings)
	r.POST("/social/twitch/commercial", sc.TwitchPlayCommercial)

	r.GET("/social/twitter/oauthurl", sc.TwitterOAuthURL)
	r.GET("/social/twitter/verify", sc.TwitterCheckForAuth)
	r.POST("/social/twitter/auth", sc.TwitterCallback)
	r.DELETE("/social/twitter/token", sc.TwitterDeleteToken)
	r.POST("/social/twitter/update", sc.TwitterSendUpdate)
	r.POST("/social/twitter/template", sc.TwitterAddTemplate)
	r.GET("/social/twitter/template", sc.TwitterGetTemplates)
	r.DELETE("/social/twitter/template/:index", sc.TwitterDeleteTemplate)
	r.PUT("/social/twitter/settings", sc.TwitterSetSettings)
	r.GET("/social/twitter/settings", sc.TwitterGetSettings)

}

// NewSocialController will return a new social controller
func NewSocialController(twitchClientID, twitchClientSecret, twitchCallback, twitterKey, twitterSecret, twitterCallback string, b *common.Controller, router *httprouter.Router) {
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

	c.registerRoutes(router)
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
		} else if i == 3 {
			sc.twitchUpdateChannelID()
		}
	}
}
