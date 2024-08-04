package auth

import (
	"context"
	"holvit/constants"
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
	return true, nil
}

func (s *VerifyEmailStep) Prepare(ctx context.Context, info *services.LoginInfo) error {
	return nil
}
