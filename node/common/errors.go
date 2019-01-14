package common

import (
	"errors"
)

var (
	// parameter is invalid
	ErrInvalidParameter = errors.New("invalid parameter")

	ErrInvalidDatetime = errors.New("invalid datetime")

	ErrInvalidTaskName = errors.New("invalid task name")

	ErrInvalidPluginName = errors.New("invalid plugin name")

	ErrNoTask = errors.New("no task, may be executed")
)
