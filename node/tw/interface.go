package tw

import "time"

type Item interface {
	Delay() time.Duration
	ID() int64
}
