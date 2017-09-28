package runs

import "github.com/onestay/MarathonTools-API/api/routes/common"

type RunController struct {
	base *common.Controller
}

func NewRunController(b *common.Controller) *RunController {
	return &RunController{
		base: b,
	}
}
