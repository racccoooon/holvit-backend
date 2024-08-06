package utils

import (
	"encoding/json"
	"holvit/h"
)

func FromRawMessage[T any](raw json.RawMessage) h.Result[T] {
	var t T
	err := json.Unmarshal(raw, &t)
	if err != nil {
		return h.Err[T](err)
	}
	return h.Ok(t)
}
