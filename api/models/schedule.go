package models

import "database/sql"

func AppendRunToSchedule(runID int64, db *sql.DB) error {
	stmt, err := db.Prepare("INSERT INTO schedule(run_id) VALUES (?)")
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(runID)
	if err != nil {
		return err
	}

	return nil
}
