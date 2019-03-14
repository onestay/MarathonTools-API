package common

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type horaroSchedule struct {
	Data struct {
		SetupT  int      `json:"setup_t"`
		Columns []string `json:"columns"`
		Items   []struct {
			Length     string    `json:"length"`
			LengthT    int       `json:"length_t"`
			Scheduled  time.Time `json:"scheduled"`
			ScheduledT int       `json:"scheduled_t"`
			Data       []string  `json:"data"`
		} `json:"items"`
	} `json:"data"`
}

func (c Controller) UpdateScheduleHTTP(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	b, err := c.updateSchedule()
	if err != nil {
		c.Response("", "Error updating schedule", http.StatusInternalServerError, w)
		c.LogError("while updating schedule", err, false)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Write(b)
}

func (c Controller) updateSchedule() ([]byte, error) {
	res, err := http.Get("https://horaro.org/-/api/v1/schedules/68111mc0ucfb2t7af9")
	if err != nil {
		return nil, err
	} else if res.StatusCode != 200 {
		return nil, errors.New("Non 200 status code returned from horaro server")
	}

	defer res.Body.Close()
	var t horaroSchedule

	json.NewDecoder(res.Body).Decode(&t)

	// scuffed website GIVES ME THE FUCKING TIME IN A SPECIFIC FORMAT BUT DOESN'T ACCEPT THAT FUCKING FORMAT WHEN IMPORTING AGAIN
	// ???????????????????????????????????????????????
	for i := 0; i < len(t.Data.Items); i++ {
		t.Data.Items[i].Length = fmtDuration(time.Duration(t.Data.Items[i].LengthT) * time.Second)
	}

	// time difference between current run end and next run scheduled time
	difference := (time.Duration(t.Data.Items[0].LengthT) * time.Second) + time.Now().Sub(t.Data.Items[1].Scheduled)

	t.Data.Items[0].Length = fmtDuration(difference)

	// actually parse the [][]string into a csv file
	var dest bytes.Buffer
	w := csv.NewWriter(&dest)
	w.WriteAll(horaroScheduleToCSV(t))

	if err := w.Error(); err != nil {
		return nil, err
	}

	return dest.Bytes(), nil
}

// https://stackoverflow.com/a/47342272
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%02d:%02d:00", h, m)
}

func horaroScheduleToCSV(t horaroSchedule) [][]string {
	csv := make([][]string, len(t.Data.Items)+1)
	headerColumn := make([]string, len(t.Data.Columns)+1)
	for i, column := range t.Data.Columns {
		headerColumn[i] = column
	}
	fmt.Println(t.Data.Columns)
	headerColumn[len(headerColumn)-1] = "length"
	csv[0] = headerColumn
	fmt.Println(headerColumn)
	for i, item := range t.Data.Items {
		tmpColumn := make([]string, len(t.Data.Columns)+1)
		for i2, dataItem := range item.Data {
			tmpColumn[i2] = dataItem
			tmpColumn[len(tmpColumn)-1] = item.Length
		}
		csv[i+1] = tmpColumn
	}

	return csv
}
