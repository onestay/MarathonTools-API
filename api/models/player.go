package models

import "database/sql"

type PlayerInfo struct {
	Id          int64  `json:"id,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Country     string `json:"country,omitempty"`
	TwitterName string `json:"twitterName,omitempty"`
	TwitchName  string `json:"twitchName,omitempty"`
	YoutubeName string `json:"youtubeName,omitempty"`
}

func AddPlayer(player PlayerInfo, db *sql.DB) (int64, error) {
	stmt, err := db.Prepare("INSERT INTO players(display_name, country, twitter_name, twitch_name, youtube_name) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	result, err := stmt.Exec(player.DisplayName, player.Country, player.TwitterName, player.TwitchName, player.YoutubeName)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func GetPlayerById(id int64, db *sql.DB) (*PlayerInfo, error) {
	var player PlayerInfo
	err := db.QueryRow("SELECT * FROM players WHERE id=?", id).Scan(&player.Id, &player.DisplayName, &player.Country, &player.TwitterName, &player.TwitchName, &player.YoutubeName)
	if err != nil {
		return nil, err
	}

	return &player, nil
}
