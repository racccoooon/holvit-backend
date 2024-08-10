//go:build embed_static

package server

import (
	_ "embed"
	"github.com/gorilla/mux"
	"holvit/server/embed"
	"holvit/static"
	"net/http"
)

func registerStatics(r *mux.Router) {
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", static.Server(embed.AuthStatic, embed.AdminStatic)))
}
