package constants

import "fmt"

const DeviceCookieName = "holvit_device_id"
const SessionCookiePattern = "holvit_%s_session"

func SessionCookieName(realmName string) string {
	return fmt.Sprintf(SessionCookiePattern, realmName)
}

const CredentialTypePassword = "password"
const CredentialTypeTotp = "totp"

const QueuedJobSendMail = "send_mail"

const ClaimMapperUserInfo = "user_info"

const UserInfoPropertyId = "id"
const UserInfoPropertyEmail = "email"
const UserInfoPropertyEmailVerified = "email_verified"
const UserInfoPropertyUsername = "username"

const HashAlgorithmBCrypt = "bcrypt"
const HashAlgorithmSCrypt = "scrypt"
const HashAlgorithmArgon2id = "argon2id"

const AuthorizationResponseModeQuery = "query"

const AuthorizationResponseTypeCode = "code"

const TokenGrantTypeAuthorizationCode = "authorization_code"
const TokenGrantTypeRefreshToken = "refresh_token"

const CodeChallengeMethodS256 = "S256"

const FrontendModeAuthenticate = "authenticate"
const FrontendModeAuthorize = "authorize"

const AuthenticateStepVerifyPassword = "verify_password"
const AuthenticateStepVerifyEmail = "verify_email"
const AuthenticateStepResetPassword = "reset_password"
const AuthenticateStepTotpOnboarding = "totp_onboarding"
const AuthenticateStepVerifyTotp = "verify_totp"
const AuthenticateStepVerifyDevice = "verify_device"
const AuthenticateStepSubmit = "submit"

const TotpSecretLength = 32

const MasterRealmName = "admin"
const SuperUserRoleName = "superuser"

const SqlErrorCodeRealmsDoNotMatch = "VV001"
