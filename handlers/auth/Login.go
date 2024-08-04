package auth

import (
	"context"
	"holvit/constants"
	"holvit/services"
	"net/http"
)

func Login(w http.ResponseWriter, r *http.Request) {
	//TODO: implement me
}

type SubmitLoginStep struct {
}

func (s *SubmitLoginStep) Name() string {
	return constants.AuthenticateStepSubmit
}

func (s *SubmitLoginStep) NeedsToRun(ctx context.Context, info *services.LoginInfo) (bool, error) {
	return true, nil
}

func (s *SubmitLoginStep) Prepare(ctx context.Context, info *services.LoginInfo) error {
	return nil
}
