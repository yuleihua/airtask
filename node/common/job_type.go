package common

import (
	"errors"
	"fmt"
	"strings"
)

// key type for JobType
type JobType int

const (
	JobTypeUnkown JobType = iota
	JobTypeCmd
	JobTypeFile
	JobTypePlugin
)

var ErrInvalidJobType = errors.New("no key type")

// UnmarshalText parses the given text into a JobType.
func (jt *JobType) UnmarshalText(data []byte) error {
	input := strings.TrimSpace(string(data))

	switch input {
	case "cmd":
		*jt = JobTypeCmd
		return nil
	case "sh":
		*jt = JobTypeFile
		return nil
	case "plugin":
		*jt = JobTypePlugin
		return nil
	}

	return ErrInvalidJobType
}

func (jt JobType) String() string {
	switch jt {
	case JobTypeCmd:
		return "cmd"
	case JobTypeFile:
		return "sh"
	case JobTypePlugin:
		return "plugin"
	}
	return fmt.Sprintf("unknown type : %d", jt)
}

func (jt JobType) MarshalText() ([]byte, error) {
	switch jt {
	case JobTypeCmd:
		return []byte("cmd"), nil
	case JobTypeFile:
		return []byte("sh"), nil
	case JobTypePlugin:
		return []byte("plugin"), nil
	}
	return nil, ErrInvalidJobType
}
