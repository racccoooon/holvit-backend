package h

import (
	"fmt"
	"runtime"
)

func unwrapErr(msg string, err error) error {
	pc, file, line, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()
	if err != nil {
		return fmt.Errorf("%s in %s (%s:%d): %w", msg, funcName, file, line, err)
	} else {
		return fmt.Errorf("%s in %s (%s:%d)", msg, funcName, file, line)
	}
}
