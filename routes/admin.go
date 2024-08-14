package routes

var AdminFrontend = SimpleRoute("/admin")

var adminApiBase = "/api/admin"

var AdminApiBase = SimpleRoute(adminApiBase)

var FindRealms = RealmRoute(adminApiBase + "/realms")
var FindUsers = RealmRoute(adminApiBase + "/realms/{realmName}/users")
var FindScopes = RealmRoute(adminApiBase + "/realms/{realmName}/scopes")
