package models

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestPlayer(t *testing.T) {
	db, err := sql.Open("sqlite3", "../../db/test.sqlite")
	if err != nil {
		t.Fail()
	}

	for i := 0; i < 10; i++ {
		p := PlayerInfo{
			Id:          0,
			DisplayName: fmt.Sprintf("onestay%d", i),
			Country:     "de",
			TwitterName: "@onest4y",
			TwitchName:  "onestay",
			YoutubeName: "",
		}

		_, err = AddPlayer(p, db)
		if err != nil {
			panic(err)
		}
	}

	player, err := GetPlayerById(1, db)
	if err != nil {
		panic(err)
	}

	if player.DisplayName != "onestay0" {
		t.Errorf("Expected DisplayName to be %s but got %s", "onestay0", player.DisplayName)
	}

	_, err = GetPlayerById(10000, db)
	if err == nil {
		t.Errorf("Expected GetPlayerById with id 10000 to fail")
	}
}
