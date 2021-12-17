package common

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-redis/redis"

	"github.com/julienschmidt/httprouter"
)

// Settings provides just some general settings
type Settings struct {
	Currency            string `json:"currency"`
	Chat                string `json:"chat"`
	SocialCircleTime    int    `json:"socialCircleTime"`
	TwitchUpdateChannel string `json:"twitchUpdateChannel"`
}

// SettingsProvider provides something idk
type SettingsProvider struct {
	S *Settings
	b *Controller
}

// InitSettings will return a SettingsProvider
func InitSettings(b *Controller) *SettingsProvider {
	s := Settings{}
	log.Println("Initializing settings...")
	if b, err := b.RedisClient.Get("settings").Bytes(); err != redis.Nil && len(b) != 0 {
		log.Println("Found settings in redis")
		json.Unmarshal(b, &s)
	} else {
		log.Println("No saved settings found. Initializing with default values")
		s.Chat = "onestay"
		s.Currency = "$"
		s.SocialCircleTime = 30000
		s.TwitchUpdateChannel = ""
	}

	return &SettingsProvider{
		S: &s,
		b: b,
	}
}

// SetSettings is used to set the settings
func (s *SettingsProvider) SetSettings(_ http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	newSettings := Settings{}

	json.NewDecoder(r.Body).Decode(&newSettings)

	s.S = &newSettings
	go func() {
		s.b.SocialUpdatesChan <- 3
	}()
	go s.b.WSSettingUpdate()
	go s.saveToRedis()
}

// GetSettings returns all settings
func (s *SettingsProvider) GetSettings(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	json.NewEncoder(w).Encode(s.S)
}

func (s *SettingsProvider) saveToRedis() {
	b, _ := json.Marshal(s.S)
	err := s.b.RedisClient.Set("settings", b, 0).Err()
	if err != nil {
		s.b.LogError("while saving settings to redis", err, false)
	}
}
