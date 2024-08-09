package routes

import (
	"holvit/config"
	"strings"
)

type RealmRoute string

func makeUrl(path string) string {
	return config.C.BaseUrl + path // TODO: make sure it is separated by a single slash, not multiple or none
}

func (r RealmRoute) Url(realmName string) string {
	return makeUrl(strings.Replace(string(r), "{realmName}", realmName, -1)) // TODO: maybe there's something better?
}

func (r RealmRoute) String() string { return string(r) }

type SimpleRoute string

func (r SimpleRoute) Url() string {
	return makeUrl(string(r))
}

func (r SimpleRoute) String() string { return string(r) }
