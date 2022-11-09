package dix

import (
	"fmt"
)

type Err struct {
	Err    error
	Msg    string
	Detail string
}

func (e Err) Unwrap() error {
	if e.Err != nil {
		return fmt.Errorf("dix unknown err: %w", e.Err)
	}

	return fmt.Errorf("dix: msg=%q detail=%q", e.Msg, e.Detail)
}

func (e Err) String() string {
	return fmt.Sprintf("dix: msg=%q err=%v detail=%q", e.Msg, e.Err, e.Detail)
}

func (e Err) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}

	return fmt.Sprintf("msg=%q detail=%q", e.Msg, e.Detail)
}
