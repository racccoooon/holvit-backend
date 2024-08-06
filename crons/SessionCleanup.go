package crons

import (
	"context"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/repos"
)

func SessionCleanup() {
	logging.Logger.Debug("Cleaning sessions...")

	scope := ioc.RootScope.NewScope()
	defer scope.Close()
	ctx := middlewares.ContextWithNewScope(context.Background(), scope)

	sessionRepository := ioc.Get[repos.SessionRepository](scope)
	sessionRepository.DeleteOldSessions(ctx)
}
