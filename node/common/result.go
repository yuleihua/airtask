package common

type Result struct {
	Id        string `json:"name"`
	BeginTime int64  `json:"begin_time"`
	EndTime   int64  `json:"end_time"`
	ErrorMsg  string `json:"error"`
	Extra     []byte `json:"output"`
}

func NewResult(id string, begin int64) *Result {
	return &Result{
		Id:        id,
		BeginTime: begin,
	}
}

func NewResultWithEnd(id string, begin, end int64, msg string, extra []byte) *Result {
	return &Result{
		Id:        id,
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
