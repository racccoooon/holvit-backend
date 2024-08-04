package crons

import (
	"context"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/repositories"
)

func SessionCleanup() {
	logging.Logger.Debug("Cleaning sessions...")

	scope := ioc.RootScope.NewScope()
	defer scope.Close()
	ctx := middlewares.ContextWithNewScope(context.Background(), scope)

	sessionRepository := ioc.Get[repositories.SessionRepository](scope)
	err := sessionRepository.DeleteOldSessions(ctx)

	if err != nil {
		logging.Logger.Error(err)
	}
}
