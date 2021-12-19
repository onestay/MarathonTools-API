package models

import (
	"database/sql"
	"fmt"
)

const (
	RunIdEmptyRun = -1
	RunIdNotInit  = -2
)

// Run represents a single run
// TODO: look into making more fields private here
type Run struct {
	Id            int64             `json:"runID"`
	GameInfo      GameInfo          `json:"gameInfo"`
	RunInfo       RunInfo           `json:"runInfo"`
	Players       []PlayerInfo      `json:"players"`
	RunTime       runTime           `json:"runTime"`
	PlayerRunTime map[int64]runTime `json:"playerRunTime"`
}

type GameInfo struct {
	GameName    string `json:"gameName"`
	ReleaseYear int    `json:"releaseYear"`
}

type RunInfo struct {
	Estimate int64  `json:"estimate"`
	Category string `json:"category"`
	Platform string `json:"platform"`
}

type runTime struct {
	Finished bool    `json:"finished"`
	Time     float64 `json:"time"`
}

func CreateRun(gi GameInfo, ri RunInfo, players []PlayerInfo) Run {
	var run Run

	run.GameInfo = gi
	run.Players = players
	run.RunInfo = ri
	run.Id = RunIdNotInit

	run.RunTime = runTime{}
	run.PlayerRunTime = make(map[int64]runTime, len(players))

	return run
}

// EmptyRun is a run identified by id 0.
func EmptyRun() *Run {
	return &Run{Id: RunIdEmptyRun}
}

func (r Run) IsEmptyRun() bool {
	return r.Id == RunIdEmptyRun
}

func (r *Run) SetID(id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid id %d", id)
	}

	r.Id = id

	return nil
}

func AddRun(run Run, db *sql.DB) (int64, error) {
	id, err := insertRunIntoDb(&run, db)
	if err != nil {
		return 0, err
	}

	err = run.SetID(id)
	if err != nil {
		return 0, err
	}

	// FIXME: if this fails we probably want to delete the run from the db too since otherwise it messes with stuff
	err = AppendRunToSchedule(id, db)
	if err != nil {
		return 0, err
	}

	return id, nil
}

//func GetRuns(db *sql.DB) []Run {
//rows, err := db.Query("SELECT * FROM run_players JOIN runs ON run_players.run_id=runs.id JOIN players ON run_players.player_id=players.id")
//}

func insertRunIntoDb(run *Run, db *sql.DB) (int64, error) {
	stmt, err := db.Prepare("INSERT INTO runs(game_name, release_year, estimate, category, platform, finished, time) VALUES (?, ?, ?, ?, ?, 0, 0)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	result, err := stmt.Exec(run.GameInfo.GameName, run.GameInfo.ReleaseYear, run.RunInfo.Estimate, run.RunInfo.Category, run.RunInfo.Platform)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	err = run.SetID(id)
	if err != nil {
		return 0, err
	}

	// FIXME: delete original entry from DB on error
	err = insertRunPlayerRelation(run, db)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func insertRunPlayerRelation(run *Run, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT INTO run_players (run_id, player_id) VALUES (?, ?)")
	if err != nil {
		return err
	}

	for _, player := range run.Players {
		_, err := stmt.Exec(run.Id, player.Id)
		if err != nil {
			// FIXME: cleanup needed
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
