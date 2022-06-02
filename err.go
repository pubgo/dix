package dix

import "fmt"

type Err struct {
	Err    error
	Msg    string
	Detail string
}

func (e Err) Unwrap() error { return e.Err }

func (e Err) String() string {
	return fmt.Sprintf("%s, err=%v detail=%s", e.Msg, e.Err, e.Detail)
}

func (e Err) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}

	if e.Msg != "" {
		return e.Msg
	}

	if e.Detail != "" {
		return e.Detail
	}

	return "dix unknown error"
}
