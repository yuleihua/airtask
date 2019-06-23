// Copyright 2018 The huayulei_2003@hotmail.com Authors
// This file is part of the airfk library.
//
// The airfk library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The airfk library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the airfk library. If not, see <http://www.gnu.org/licenses/>.
package common

import (
	"fmt"
	"time"

	"airman.com/airfk/pkg/common/hexutil"
)

//go:generate gencodec -type Job -field-override jobMarshaling -out gen_job_json.go

// Job is task job.
type Job struct {
	Name      string  `json:"name"     gencodec:"required"`
	Type      JobType `json:"type"`
	UUID      ItemID  `json:"uuid"`
	Retry     int     `json:"retry"    gencodec:"required"`
	Interval  int     `json:"interval" gencodec:"required"`
	AddTime   int64   `json:"add_time"`
	LimitTime int64   `json:"limit_time"`
	Extra     []byte  `json:"extra"`
}

type jobMarshaling struct {
	Extra hexutil.Bytes
	//UUID hexutil.Uint64
}

func NewJob(uuid uint64, name string, t JobType, delay, retry int) *Job {
	duration := delay
	if duration < 1 {
		duration = 1
	}
	r := retry
	if retry == 0 {
		r = 1
	}
	return &Job{
		Interval: duration,
		Name:     name,
		UUID:     EncodeItemID(uuid),
		Type:     t,
		Retry:    r,
	}
}

func (j *Job) Delay() time.Duration {
	return time.Duration(j.Interval) * time.Second
}

func (j *Job) ID() int64 {
	return int64(j.UUID.Uint64())
}

func (j *Job) String() string {
	return fmt.Sprintf("id:%d,name:%s,delay:%v,retry:%d,create:%d,limit:%d",
		j.UUID, j.Name, j.Interval, j.Retry, j.AddTime, j.LimitTime)
}
