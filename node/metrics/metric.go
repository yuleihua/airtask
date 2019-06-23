package metrics

import (
	metrics "github.com/rcrowley/go-metrics"
)

var (
	TaskAddMeter     = metrics.NewRegisteredMeter("task/add", nil)
	TaskExecuteMeter = metrics.NewRegisteredMeter("task/execute", nil)
	TaskExecuteTimer = metrics.NewRegisteredTimer("task/useTime", nil)
)
