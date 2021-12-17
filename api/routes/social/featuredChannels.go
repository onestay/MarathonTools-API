package social

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const FeaturedChannelsUrl = "https://api.furious.pro/featuredchannels/bot"

func (sc Controller) UpdateFeaturedChannels() error {
	if sc.featuredChannelsKey == "" {
		return nil
	}

	players := make([]string, len(sc.base.CurrentRun.Players))

	for i, player := range sc.base.CurrentRun.Players {
		if len(player.TwitchName) != 0 {
			players[i] = player.TwitchName
		} else {
			players[i] = player.DisplayName
		}
	}

	playersString := strings.Join(players, ",")

	reqUrl, err := url.Parse(FeaturedChannelsUrl + "/" + sc.featuredChannelsKey + "/" + playersString)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", reqUrl.String(), nil)
	if err != nil {
		return err
	}

	res, err := sc.base.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("non 200 returned from updatedFeaturedChannels")
	}

	return nil
}
