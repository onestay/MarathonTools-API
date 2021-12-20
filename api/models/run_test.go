package models

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestAddRun(t *testing.T) {
	db, err := sql.Open("sqlite3", "../../db/test.sqlite")
	if err != nil {
		panic(err)
	}

	gi1 := GameInfo{GameName: "Portal 1"}
	gi2 := GameInfo{GameName: "Portal 2"}
	gi3 := GameInfo{GameName: "Portal 3"}
	gi4 := GameInfo{GameName: "Portal 4"}
	gi5 := GameInfo{GameName: "Portal 5"}

	p1, _ := GetPlayerById(1, db)
	p2, _ := GetPlayerById(2, db)
	p3, _ := GetPlayerById(3, db)
	p4, _ := GetPlayerById(4, db)
	p5, _ := GetPlayerById(5, db)
	p6, _ := GetPlayerById(6, db)
	p7, _ := GetPlayerById(7, db)
	p8, _ := GetPlayerById(8, db)
	p9, _ := GetPlayerById(9, db)

	var pi1 = []*PlayerInfo{p1}
	var pi2 = []*PlayerInfo{p2, p6}
	var pi3 = []*PlayerInfo{p4, p5}
	var pi4 = []*PlayerInfo{p8, p1, p9, p3}
	var pi5 = []*PlayerInfo{p7}

	AddRun(CreateRun(gi1, RunInfo{}, pi1), db)
	AddRun(CreateRun(gi2, RunInfo{}, pi2), db)
	AddRun(CreateRun(gi3, RunInfo{}, pi3), db)
	AddRun(CreateRun(gi4, RunInfo{}, pi4), db)
	AddRun(CreateRun(gi5, RunInfo{}, pi5), db)

	runs, err := GetRuns(db)
	if err != nil {
		t.Fatalf("Error %v", err)
	}

	if len(runs) != 5 {
		t.Fatalf("invalid run length expected 5 but got %d", len(runs))
	}

	aRun, err := getRunBySchedulePosition(1, db)
	if err != nil {
		panic(err)
	}

	if aRun.GameInfo.GameName != "Portal 1" {
		t.Fatalf("Expected game name at schedule position 1 to be \"Portal3\" but got %s", aRun.GameInfo.GameName)
	}

}
