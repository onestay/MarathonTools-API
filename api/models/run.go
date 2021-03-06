package models

import (
	"gopkg.in/mgo.v2/bson"
)

// Run represents a single run
type Run struct {
	RunID    bson.ObjectId `json:"runID" bson:"_id"`
	GameInfo GameInfo      `json:"gameInfo" bson:"gameInfo"`
	RunInfo  runInfo       `json:"runInfo" bson:"runInfo"`
	Players  []PlayerInfo  `json:"players" bson:"playerInfo"`
}

type GameInfo struct {
	GameName    string `json:"gameName" bson:"gameName"`
	ReleaseYear int    `json:"releaseYear" bson:"releaseYear"`
}

type runInfo struct {
	Estimate string `json:"estimate" bson:"estimate"`
	Category string `json:"category" bson:"category"`
	Platform string `json:"platform" bso:"platform"`
}

type PlayerInfo struct {
	DisplayName string          `json:"displayName" bson:"displayName"`
	Country     string          `json:"country" bson:"country"`
	TwitterName string          `json:"twitterName" bson:"twitterName"`
	TwitchName  string          `json:"twitchName" bson:"twitchName"`
	YoutubeName string          `json:"youtubeName" bson:"youtubeName"`
	Timer       timerPlayerInfo `json:"timer" bson:"timer"`
}

type timerPlayerInfo struct {
	Finished bool    `json:"finished" bson:"finished"`
	Time     float64 `json:"time" bson:"time"`
}
