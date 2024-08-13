package routes

const realmApiBase = "/api/realms/{realmName}"

var RealmApiBase = RealmRoute(realmApiBase)

var ApiVerifyPassword = RealmRoute(realmApiBase + "/auth/verify-password")
var ApiResetPassword = RealmRoute(realmApiBase + "/auth/reset-password")
var ApiTotpOnboarding = RealmRoute(realmApiBase + "/auth/totp-onboarding")
var ApiVerifyTotp = RealmRoute(realmApiBase + "/auth/verify-totp")
var ApiVerifyDevice = RealmRoute(realmApiBase + "/auth/verify-device")
var ApiGetOnboardingTotp = RealmRoute(realmApiBase + "/auth/get-onboarding-totp")
var ApiResendEmailVerification = RealmRoute(realmApiBase + "/auth/resend-email-verification")

var LoginComplete = RealmRoute("/auth/{realmName}/login-complete")
var AuthorizeGrant = RealmRoute("/auth/{realmName}/authorize-grant")
var AuthVerifyEmail = RealmRoute("/auth/{realmName}/verify-email")
