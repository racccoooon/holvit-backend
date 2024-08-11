package routes

var OidcAuthorize = RealmRoute("/oidc/{realmName}/authorize")
var OidcToken = RealmRoute("/oidc/{realmName}/token")
var OidcUserInfo = RealmRoute("/oidc/{realmName}/userinfo")
var OidcJwks = RealmRoute("/oidc/{realmName}/jwks")
var OidcLogout = RealmRoute("/oidc/{realmName}/logout")
var WellKnown = RealmRoute("/oidc/{realmName}/.well-known/openid-configuration")
