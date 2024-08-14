package routes

var AdminFrontend = SimpleRoute("/admin")

var adminApiBase = "/api/admin"

var AdminApiBase = SimpleRoute(adminApiBase)

var FindUsers = RealmRoute(adminApiBase + "/realms/{realmName}/users")
