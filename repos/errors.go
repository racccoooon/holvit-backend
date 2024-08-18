package repos

import (
	"github.com/lib/pq"
	"holvit/constants"
)

type RealmsDoNotMatchError struct{}

func (RealmsDoNotMatchError) Error() string {
	return "Realms do not match"
}

func mapCustomErrorCodes(err error) error {
	if err == nil {
		return nil
	}

	if pqErr, ok := err.(*pq.Error); ok {
		if pqErr.Code == constants.SqlErrorCodeRealmsDoNotMatch {
			return RealmsDoNotMatchError{}
		}
	}

	return err
}
