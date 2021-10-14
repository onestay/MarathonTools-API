package social

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/onestay/MarathonTools-API/api/common"

	"github.com/dghubble/oauth1"
	"github.com/dghubble/oauth1/twitter"
)

// Controller holds all the info and methods
type Controller struct {
	twitchInfo          *twitchInfo
	twitterInfo         *oauth1.Config
	base                *common.Controller
	socialAuth          *socialAuthInfo
	featuredChannelsKey string
}

type twitchInfo struct {
	ClientID     string
	ClientSecret string
	Scope        string
	RedirectURI  string
}

type socialAuthAvailResponse struct {
	Twitter bool `json:"twitter,omitempty"`
	Twitch  bool `json:"twitch,omitempty"`
}

func (sc Controller) checkSocialAuth() (*socialAuthAvailResponse, error) {
	url := sc.socialAuth.url + "/api/v1/avail"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", sc.socialAuth.key)

	res, err := sc.base.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	avail := socialAuthAvailResponse{}

	err = json.NewDecoder(res.Body).Decode(&avail)
	if err != nil {
		return nil, err
	}

	return &avail, nil
}

type socialAuthInfo struct {
	url string
	key string
}

type SocialAuthErrorResponse struct {
	Status  int    `json:"status,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

func (sc Controller) registerRoutes(r *httprouter.Router) {
	r.GET("/social/twitch/verify", sc.TwitchCheckForAuth)
	r.DELETE("/social/twitch/token", sc.TwitchDeleteToken)
	r.GET("/social/twitch/executetemplate", sc.TwitchExecuteTemplate)
	r.PUT("/social/twitch/update", sc.TwitchUpdateInfo)
	r.PUT("/social/twitch/settings", sc.TwitchSetSettings)
	r.GET("/social/twitch/settings", sc.TwitchGetSettings)
	r.POST("/social/twitch/commercial", sc.TwitchPlayCommercial)

	r.GET("/social/twitter/verify", sc.TwitterCheckForAuth)
	r.DELETE("/social/twitter/token", sc.TwitterDeleteToken)
	r.POST("/social/twitter/update", sc.TwitterSendUpdate)
	r.POST("/social/twitter/template", sc.TwitterAddTemplate)
	r.GET("/social/twitter/template", sc.TwitterGetTemplates)
	r.DELETE("/social/twitter/template/:index", sc.TwitterDeleteTemplate)
	r.PUT("/social/twitter/settings", sc.TwitterSetSettings)
	r.GET("/social/twitter/settings", sc.TwitterGetSettings)

}

// NewSocialController will return a new social controller
func NewSocialController(twitchClientID, twitchClientSecret, twitchCallback, twitterKey, twitterSecret, twitterCallback, socialAuthURL, socialAuthKey, featuredChannelsKey string, b *common.Controller, router *httprouter.Router) {
	t := &twitchInfo{
		ClientID:     twitchClientID,
		ClientSecret: twitchClientSecret,
		Scope:        "channel:edit:commercial channel:manage:broadcast",
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
		socialAuth: &socialAuthInfo{
			url: socialAuthURL,
			key: socialAuthKey,
		},
		featuredChannelsKey: featuredChannelsKey,
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
		} else if i == 0x50 {
			err := sc.UpdateFeaturedChannels()
			if err != nil {
				sc.base.LogError("while updating featured channels", err, true)
			}
		}
	}
}
