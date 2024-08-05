package auth

import (
	"context"
	"holvit/constants"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/services"
	"net/http"
)

func VerifyEmail(w http.ResponseWriter, r *http.Request) {
	//TODO: implement
}

type VerifyEmailStep struct {
}

func (s *VerifyEmailStep) Name() string {
	return constants.AuthenticateStepVerifyEmail
}

func (s *VerifyEmailStep) NeedsToRun(ctx context.Context, info *services.LoginInfo) (bool, error) {
	scope := middlewares.GetScope(ctx)

	userRepository := ioc.Get[repos.UserRepository](scope)
	user := userRepository.FindUserById(ctx, info.UserId).Unwrap()

	if user.Email.IsSome() && user.EmailVerified {
		return true, nil
	}

	return false, nil
}

func (s *VerifyEmailStep) Prepare(ctx context.Context, info *services.LoginInfo) error {
	return nil
}
