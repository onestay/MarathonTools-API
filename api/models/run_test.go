package models

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestNewMarathon(t *testing.T) {
	db, err := sql.Open("sqlite3", "../../db/test.sqlite")
	if err != nil {
		panic(err)
	}

	gi := GameInfo{
		GameName:    "Portal 3",
		ReleaseYear: 2025,
	}

	ri := RunInfo{
		Estimate: 30,
		Category: "100%",
		Platform: "PC",
	}

	p1, err := GetPlayerById(1, db)
	p2, err := GetPlayerById(2, db)
	p3, err := GetPlayerById(3, db)

	var pi []*PlayerInfo
	pi = append(pi, p1)
	pi = append(pi, p2)

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

	var pi2 []*PlayerInfo
	pi2 = append(pi2, p3)
	_, err = AddRun(CreateRun(gi, ri, pi2), db)
	if err != nil {
		panic(err)
	}

	runs, err := GetRuns(db)
	if err != nil {
		t.Errorf("Error %v", err)
	}

	if len(runs) == 0 {
		t.Errorf("invalid run length")
	}
}
