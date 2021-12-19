package marathon

import "github.com/onestay/MarathonTools-API/api/models"

type RunState struct {
	index   int32
	current *models.Run
	next    *models.Run
	prev    *models.Run
	upNext  *models.Run
}

func (m Marathon) GetState() RunState {
	return *m.runState
}

func (m Marathon) SetCurrentRun(run *models.Run) {
	m.marathonMutex.Lock()
	m.runState.current = run
	m.marathonMutex.Unlock()
}

func (m Marathon) SetNextRun(run *models.Run) {
	m.marathonMutex.Lock()
	m.runState.next = run
	m.marathonMutex.Unlock()
}

func (m Marathon) SetPrevRun(run *models.Run) {
	m.marathonMutex.Lock()
	m.runState.prev = run
	m.marathonMutex.Unlock()
}

func (m Marathon) SetUpNextRun(run *models.Run) {
	m.marathonMutex.Lock()
	m.runState.upNext = run
	m.marathonMutex.Unlock()
}
