package marathon

import (
	"github.com/onestay/MarathonTools-API/api/models"
	"sync"
)

const InitialRunCap = 30

// Marathon represents the general Marathon
type Marathon struct {
	name          string
	runState      *RunState
	marathonMutex *sync.Mutex
}

func NewMarathon(name string, index int32) *Marathon {
	marathon := Marathon{
		name:          name,
		marathonMutex: &sync.Mutex{},
		runState: &RunState{
			index:   index,
			current: models.EmptyRun(),
			next:    models.EmptyRun(),
			prev:    models.EmptyRun(),
			upNext:  models.EmptyRun(),
		},
	}

	return &marathon
}
