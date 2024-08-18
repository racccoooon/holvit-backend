package requestContext

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"holvit/events"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/utils"
)

type AfterTxEventArgs struct {
	Commit   bool
	Rollback bool
}

type RequestContextService interface {
	Errors() []error
	Error(err error)
	GetTx() (*sql.Tx, error)
	Close() error
	OnAfterTx(handler events.EventHandler[AfterTxEventArgs])
}

type RequestContextServiceImpl struct {
	id string

	scope  *ioc.DependencyProvider
	errors []error

	afterTxEvent *events.Event[AfterTxEventArgs]

	tx *sql.Tx
}

func NewRequestContextService(scope *ioc.DependencyProvider) RequestContextService {
	service := RequestContextServiceImpl{
		id:           uuid.New().String(),
		scope:        scope,
		errors:       []error{},
		afterTxEvent: events.NewEvent[AfterTxEventArgs](),
	}

	return &service
}

func (rcs *RequestContextServiceImpl) OnAfterTx(handler events.EventHandler[AfterTxEventArgs]) {
	events.Subscribe(rcs.afterTxEvent, handler)
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
	var err error = nil
	args := AfterTxEventArgs{}

	if rcs.tx != nil {
		if len(rcs.errors) == 0 {
			err = rcs.tx.Commit()
			args.Commit = true
		} else {
			err = rcs.tx.Rollback()
			args.Rollback = true
		}
	}

	events.Publish(rcs.afterTxEvent, args)
	return err
}

func RunWithScope(dp *ioc.DependencyProvider, ctx context.Context, run func(ctx context.Context)) {
	scope := dp.NewScope()
	defer func() {
		err := recover()
		if err != nil {
			panic(err)
		} else {
			utils.PanicOnErr(scope.Close)
		}
	}()
	run(middlewares.ContextWithNewScope(ctx, scope))
}
