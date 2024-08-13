package routes

var ApiHealth = SimpleRoute("/api/health")

var FindUsers = RealmRoute(realmApiBase + "/api/users")
