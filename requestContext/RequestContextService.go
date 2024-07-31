package requestContext

import (
	"database/sql"
	"github.com/google/uuid"
	"holvit/ioc"
)

type RequestContextService interface {
	Errors() []error
	Error(err error)
	GetTx() (*sql.Tx, error)
	Close() error
}
type RequestContextServiceImpl struct {
	id string

	scope  *ioc.DependencyProvider
	errors []error

	tx *sql.Tx
}

func NewRequestContextService(scope *ioc.DependencyProvider) RequestContextService {
	return &RequestContextServiceImpl{
		id:     uuid.New().String(),
		scope:  scope,
		errors: []error{},
	}
}

func (rcs *RequestContextServiceImpl) Errors() []error {
	return rcs.errors
}

func (rcs *RequestContextServiceImpl) Error(err error) {
	rcs.errors = append(rcs.errors, err)
}

func (rcs *RequestContextServiceImpl) GetTx() (*sql.Tx, error) {
	if rcs.tx != nil {
		return rcs.tx, nil
	}

	db := ioc.Get[*sql.DB](rcs.scope)
	tx, err := db.Begin()

	rcs.tx = tx

	return tx, err
}

func (rcs *RequestContextServiceImpl) Close() error {
	if len(rcs.errors) == 0 {
		if rcs.tx != nil {
			return rcs.tx.Commit()
		}
	} else {
		if rcs.tx != nil {
			return rcs.tx.Rollback()
		}
	}

	return nil
}
