package models

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestNewMarathon(t *testing.T) {
	gi := GameInfo{
		GameName:    "Portal 3",
		ReleaseYear: 2025,
	}

	ri := RunInfo{
		Estimate: 30,
		Category: "100%",
		Platform: "PC",
	}

	p := PlayerInfo{
		Id:          2,
		DisplayName: "Onestay",
		Country:     "de",
		TwitterName: "",
		TwitchName:  "",
		YoutubeName: "",
	}

	var pi []PlayerInfo
	pi = append(pi, p)

	db, err := sql.Open("sqlite3", "../../db/test.sqlite")
	if err != nil {
		panic(err)
	}
	_, err = AddRun(CreateRun(gi, ri, pi), db)
	if err != nil {
		panic(err)
	}

	gi = GameInfo{
		GameName:    "Portal 4",
		ReleaseYear: 2027,
	}

	ri = RunInfo{
		Estimate: 30,
		Category: "100%",
		Platform: "PC",
	}

	p = PlayerInfo{
		DisplayName: "Onestay",
		Country:     "de",
		TwitterName: "",
		TwitchName:  "",
		YoutubeName: "",
	}

	_, err = AddRun(CreateRun(gi, ri, pi), db)
	if err != nil {
		panic(err)
	}
}
