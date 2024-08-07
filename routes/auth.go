package routes

var ApiHealth = SimpleRoute("/api/health")

const apiBase = "/api/realms/{realmName}"

var ApiBase = RealmRoute(apiBase)

var ApiVerifyPassword = RealmRoute(apiBase + "/auth/verify-password")
var ApiResetPassword = RealmRoute(apiBase + "/auth/reset-password")
var ApiTotpOnboarding = RealmRoute(apiBase + "/auth/totp-onboarding")
var ApiVerifyTotp = RealmRoute(apiBase + "/auth/verify-totp")
var ApiVerifyDevice = RealmRoute(apiBase + "/auth/verify-device")
var ApiGetOnboardingTotp = RealmRoute(apiBase + "/auth/get-onboarding-totp")
var ApiResendEmailVerification = RealmRoute(apiBase + "/auth/resend-email-verification")

var LoginComplete = RealmRoute("/auth/{realmName}/login-complete")
var AuthorizeGrant = RealmRoute("/auth/{realmName}/authorize-grant")
var AuthVerifyEmail = RealmRoute("/auth/{realmName}/verify-email")
