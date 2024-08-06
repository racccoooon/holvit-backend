package routes

var ApiHealth = SimpleRoute("/api/health")

var ApiAuthorizeGrant = RealmRoute("/api/auth/{realmName}/authorize-grant")
var ApiVerifyPassword = RealmRoute("/api/auth/{realmName}/verify-password")
var ApiVerifyEmail = RealmRoute("/api/auth/{realmName}/verify-email")
var ApiResetPassword = RealmRoute("/api/auth/{realmName}/reset-password")
var ApiTotpOnboarding = RealmRoute("/api/auth/{realmName}/totp-onboarding")
var ApiVerifyTotp = RealmRoute("/api/auth/{realmName}/verify-totp")
var ApiVerifyDevice = RealmRoute("/api/auth/{realmName}/verify-device")
var ApiLogin = RealmRoute("/api/auth/{realmName}/login")
var ApiGetOnboardingTotp = RealmRoute("/api/auth/{realmName}/get-onboarding-totp")
var ApiResendEmailVerification = RealmRoute("/api/auth/{realmName}/resend-email-verification")
