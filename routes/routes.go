package routes

import "strings"

type RealmRoute string

func (r RealmRoute) Url(realmName string) string {
	// TODO: url should also include domain etc?
	return strings.Replace(string(r), "{realmName}", realmName, -1) // TODO: maybe there's something better?
}

func (r RealmRoute) String() string { return string(r) }

type SimpleRoute string

func (r SimpleRoute) Url() string {
	return string(r) // TODO: should also include domain etc?
}

func (r SimpleRoute) String() string { return string(r) }
