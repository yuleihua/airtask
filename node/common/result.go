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
	"airman.com/airfk/pkg/common/hexutil"
	"fmt"
)

//go:generate gencodec -type Result -field-override resultMarshaling -out gen_result_json.go

// Result is result of execute task job.
type Result struct {
	ID        int64  `json:"id"          gencodec:"required"`
	BeginTime int64  `json:"begin_time"  gencodec:"required"`
	EndTime   int64  `json:"end_time"    gencodec:"required"`
	ErrorMsg  string `json:"error"       gencodec:"required"`
	Extra     []byte `json:"output"`
}

type resultMarshaling struct {
	Extra hexutil.Bytes
	//UUID hexutil.Uint64
}

func NewResult(id int64, begin int64) *Result {
	return &Result{
		ID:        id,
		BeginTime: begin,
	}
}

func NewResultWithEnd(id int64, begin, end int64, msg string, extra []byte) *Result {
	return &Result{
		ID:        id,
		BeginTime: begin,
		EndTime:   end,
		ErrorMsg:  msg,
		Extra:     extra,
	}
}

func (r *Result) Set(end int64, msg string, extra []byte) {
	if r != nil {
		r.EndTime = end
		r.ErrorMsg = msg
		r.Extra = extra
	}
}

func (r *Result) String() string {
	if r != nil {
		return fmt.Sprintf("ID:%d, begin:%d, end:%v, error:%s, extra:%v",
			r.ID, r.BeginTime, r.EndTime, r.ErrorMsg, r.Extra)
	}
	return ""
}
