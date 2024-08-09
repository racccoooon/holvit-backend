package auth

import (
	"context"
	"holvit/constants"
	"holvit/services"
)

type NextAuthenticationStep interface {
	Name() string
	NeedsToRun(ctx context.Context, info *services.LoginInfo) (bool, error)
	Prepare(ctx context.Context, info *services.LoginInfo) error
}

func getNextStep(ctx context.Context, currentStep string, info *services.LoginInfo) (NextAuthenticationStep, error) {
	var nextStep NextAuthenticationStep
	// TODO: gwen would really like to make this less brittle and more understandable
	switch currentStep {
	case constants.AuthenticateStepVerifyPassword:
		nextStep = &VerifyEmailStep{}
	case constants.AuthenticateStepVerifyEmail:
		nextStep = &ResetPasswordStep{}
	case constants.AuthenticateStepResetPassword:
		nextStep = &TotpOnboardingStep{}
	case constants.AuthenticateStepTotpOnboarding:
		nextStep = &VerifyDeviceStep{}
	case constants.AuthenticateStepVerifyTotp:
		nextStep = &VerifyDeviceStep{}
	default:
		nextStep = &SubmitLoginStep{}
	}

	needsToRun, err := nextStep.NeedsToRun(ctx, info)
	if err != nil {
		return nil, err
	}
	if !needsToRun {
		return getNextStep(ctx, nextStep.Name(), info)
	}

	return nextStep, nil
}
