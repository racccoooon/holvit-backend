//go:build !embed_static

package server

import "github.com/gorilla/mux"

func registerStatics(r *mux.Router) {}
