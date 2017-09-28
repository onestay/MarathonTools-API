package models

// Marathon represents the general Marathon
type Marathon struct {
	Name       string `json:"name" bson:"name"`
	Runs       []Run  `json:"runs" bson:"runs"`
	RunCount   int    `json:"runCount" bson:"runCount"`
	CurrentRun string `json:"currentRun" bson:"currentRun"`
	IsRunning  bool   `bson:"isRunning"`
}
