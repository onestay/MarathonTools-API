package social

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const FEATURED_CHANNELS_URL = "https://api.furious.pro/featuredchannels/bot"

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

	players_string := strings.Join(players, ",")

	req_url, err := url.Parse(FEATURED_CHANNELS_URL + "/" + sc.featuredChannelsKey + "/" + players_string)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", req_url.String(), nil)
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
