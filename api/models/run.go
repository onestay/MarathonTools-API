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
	Id            int64              `json:"runID"`
	GameInfo      GameInfo           `json:"gameInfo"`
	RunInfo       RunInfo            `json:"runInfo"`
	Players       []*PlayerInfo      `json:"players"`
	RunTime       runTime            `json:"runTime"`
	PlayerRunTime map[int64]*runTime `json:"playerRunTime"`
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

func CreateRun(gi GameInfo, ri RunInfo, players []*PlayerInfo) Run {
	var run Run

	run.GameInfo = gi
	run.Players = players
	run.RunInfo = ri
	run.Id = RunIdNotInit

	run.RunTime = runTime{}
	run.PlayerRunTime = make(map[int64]*runTime, len(players))

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
	stmt, err := tx.Prepare("INSERT INTO run_players (run_id, player_id, finished, time) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}

	for _, player := range run.Players {
		_, err := stmt.Exec(run.Id, player.Id, false, 0)
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

func GetRuns(db *sql.DB) ([]*Run, error) {
	sqlStmt := `	SELECT game_name, release_year, estimate, category, platform, r.finished, r.time, r.id, p.id, display_name, country, twitter_name, twitch_name, youtube_name, run_players.finished, run_players.time FROM run_players
					INNER JOIN runs r ON r.id=run_players.run_id
					INNER JOIN players p on p.id = run_players.player_id
					INNER JOIN schedule s on r.id = s.run_id
					ORDER BY row_number() OVER ()`
	rows, err := db.Query(sqlStmt)
	if err != nil {
		return nil, err
	}
	var runs []*Run

	var prevRunId int64 = -1
	var prevRun *Run = nil
	for rows.Next() {
		var gi GameInfo
		var ri RunInfo
		var pi PlayerInfo
		var ti runTime
		var playerTimeInfo runTime
		var run Run
		var runId int64
		err = rows.Scan(&gi.GameName, &gi.ReleaseYear, &ri.Estimate, &ri.Category, &ri.Platform, &ti.Finished, &ti.Time, &runId, &pi.Id, &pi.DisplayName, &pi.Country, &pi.TwitterName, &pi.TwitchName, &pi.YoutubeName, &playerTimeInfo.Finished, &playerTimeInfo.Time)
		if err != nil {
			return nil, err
		}

		if prevRunId == runId {
			prevRun.Players = append(prevRun.Players, &pi)
			prevRun.PlayerRunTime[pi.Id] = &playerTimeInfo
		} else {
			run.Id = runId
			run.GameInfo = gi
			run.RunInfo = ri
			run.Players = make([]*PlayerInfo, 1)
			run.PlayerRunTime = make(map[int64]*runTime)
			run.PlayerRunTime[pi.Id] = &playerTimeInfo
			run.Players[0] = &pi
			run.RunTime = ti

			prevRun = &run
			prevRunId = runId

			runs = append(runs, &run)
		}
	}

	return runs, nil
}

func getRunByRunID(runId int64, db *sql.DB) (*Run, error) {
	sqlStmt := `	SELECT game_name, release_year, estimate, category, platform, r.finished, r.time, r.id, p.id, display_name, country, twitter_name, twitch_name, youtube_name, run_players.finished, run_players.time FROM run_players
					INNER JOIN runs r ON r.id=run_players.run_id
					INNER JOIN players p on p.id = run_players.player_id
					WHERE r.id=?`
	rows, err := db.Query(sqlStmt, runId)
	if err != nil {
		return nil, err
	}

	var run *Run = nil
	for rows.Next() {
		var gi GameInfo
		var ri RunInfo
		var pi PlayerInfo
		var ti runTime
		var playerTimeInfo runTime
		var currentRun Run
		var runId int64
		err = rows.Scan(&gi.GameName, &gi.ReleaseYear, &ri.Estimate, &ri.Category, &ri.Platform, &ti.Finished, &ti.Time, &runId, &pi.Id, &pi.DisplayName, &pi.Country, &pi.TwitterName, &pi.TwitchName, &pi.YoutubeName, &playerTimeInfo.Finished, &playerTimeInfo.Time)
		if err != nil {
			return nil, err
		}

		if run != nil {
			run.Players = append(currentRun.Players, &pi)
			run.PlayerRunTime[pi.Id] = &playerTimeInfo
		} else {
			currentRun.Id = runId
			currentRun.GameInfo = gi
			currentRun.RunInfo = ri
			currentRun.Players = make([]*PlayerInfo, 1)
			currentRun.PlayerRunTime = make(map[int64]*runTime)
			currentRun.PlayerRunTime[pi.Id] = &playerTimeInfo
			currentRun.Players[0] = &pi
			currentRun.RunTime = ti

			run = &currentRun
		}
	}

	return run, nil

}

func getSchedulePositionFromRunId(runId int64, db *sql.DB) (int64, error) {
	if runId < 0 {
		return 0, fmt.Errorf("invalid runId provided")
	}
	sqlStmt := `SELECT schedule.pos FROM schedule WHERE schedule.run_id == ?`

	var currentPos int64
	err := db.QueryRow(sqlStmt, runId).Scan(&currentPos)
	if err != nil {
		return 0, err
	}

	return currentPos, nil
}

func getRunBySchedulePosition(position int64, db *sql.DB) (*Run, error) {
	sqlStmt := `SELECT schedule.run_id FROM schedule WHERE pos=?`

	var runId int64
	err := db.QueryRow(sqlStmt, position).Scan(&runId)
	if err != nil {
		return nil, err
	}

	return getRunByRunID(position, db)
}

func GetNextRun(runId int64, db *sql.DB) (*Run, error) {
	currentPos, err := getSchedulePositionFromRunId(runId, db)
	if err != nil {
		return nil, err
	}

	return getRunBySchedulePosition(currentPos+1, db)
}

func GetPrevRun(runId int64, db *sql.DB) (*Run, error) {
	currentPos, err := getSchedulePositionFromRunId(runId, db)
	if err != nil {
		return nil, err
	}

	return getRunBySchedulePosition(currentPos-1, db)
}

func DeleteRun(runId int64, db *sql.DB) error {
	_, err := db.Exec("DELETE FROM schedule WHERE run_id=?", runId)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM run_players WHERE run_id=?", runId)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM runs WHERE id=?", runId)
	if err != nil {
		return err
	}

	return nil
}
