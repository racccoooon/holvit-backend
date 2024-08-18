package crons

import (
	"context"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/requestContext"
)

func SessionCleanup() {
	requestContext.RunWithScope(ioc.RootScope, context.Background(), func(ctx context.Context) {
		logging.Logger.Debug("Cleaning sessions...")
		scope := middlewares.GetScope(ctx)

		sessionRepository := ioc.Get[repos.SessionRepository](scope)
		sessionRepository.DeleteOldSessions(ctx)
	})
}
