package handlers

import (
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/services"
	"net/http"
)

func AdminFrontend(w http.ResponseWriter, r *http.Request) {
	scope := middlewares.GetScope(r.Context())
	frontendService := ioc.Get[services.FrontendService](scope)
	frontendService.WriteAdminFrontend(w)
}
