package services

import (
	"encoding/json"
	"holvit/config"
	"holvit/routes"
	"holvit/services/generated"
	"net/http"
)

type FrontendService interface {
	WriteAuthFrontend(w http.ResponseWriter, realmName string, frontendData AuthFrontendData)
}

func NewFrontendService() FrontendService {
	var scripts []Script
	var styles []string

	if config.C.IsDevelopment() {
		if config.C.Development.AuthFrontendUrl != "" {
			scripts = append(scripts, Script{Url: config.C.Development.AuthFrontendUrl + "@id/virtual:vue-devtools-path:overlay.js", Type: "module"})
			scripts = append(scripts, Script{Url: config.C.Development.AuthFrontendUrl + "@id/virtual:vue-inspector-path:load.js", Type: "module"})
			scripts = append(scripts, Script{Url: config.C.Development.AuthFrontendUrl + "@vite/client", Type: "module"})
			scripts = append(scripts, Script{Url: config.C.Development.AuthFrontendUrl + "src/main.js", Type: "module"})
		} else {
			scripts = append(scripts, Script{Url: "/static/" + generated.AuthManifest.JsEntrypoint})
			for _, name := range generated.AuthManifest.Stylesheets {
				styles = append(styles, "/static/"+name)
			}
		}
	} else {
		scripts = append(scripts, Script{Url: config.C.StaticRoot + generated.AuthManifest.JsEntrypoint})
		for _, name := range generated.AuthManifest.Stylesheets {
			styles = append(styles, config.C.StaticRoot+name)
		}
	}
	return &frontendServiceImpl{
		authPage: Page{
			Title:       "",
			Scripts:     scripts,
			Stylesheets: styles,
		},
	}
}

type frontendServiceImpl struct {
	authPage Page
}

func (f *frontendServiceImpl) WriteAuthFrontend(w http.ResponseWriter, realmName string, frontendData AuthFrontendData) {
	page := f.authPage
	page.Title = "TODO"
	page.JsonData = map[string]interface{}{
		"authInfo": frontendData,
		"apiBase":  routes.ApiBase.Url(realmName),
	}
	writePage(w, page)
}

type AuthFrontendUser struct {
	Name string `json:"name"`
}

type AuthFrontendScope struct {
	Required    bool   `json:"required"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

type AuthFrontendDataAuthorize struct {
	ClientName string              `json:"clientName"`
	User       AuthFrontendUser    `json:"user"`
	Scopes     []AuthFrontendScope `json:"scopes"`
	Token      string              `json:"token"`
	RefuseUrl  string              `json:"refuseUrl"`
	LogoutUrl  string              `json:"logoutUrl"`
	GrantUrl   string              `json:"grantUrl"`
}

type AuthFrontendDataAuthenticate struct {
	ClientName       string `json:"clientName"`
	Token            string `json:"token"`
	UseRememberMe    bool   `json:"useRememberMe"`
	RegisterUrl      string `json:"registerUrl"`
	LoginCompleteUrl string `json:"loginCompleteUrl"`
}

type AuthFrontendData struct {
	Mode         string                        `json:"mode"`
	Authorize    *AuthFrontendDataAuthorize    `json:"authorize"`
	Authenticate *AuthFrontendDataAuthenticate `json:"authenticate"`
}

type Script struct {
	Url  string
	Type string
}

type Page struct {
	Title       string
	Scripts     []Script
	Stylesheets []string
	JsonData    map[string]interface{}
}

func writeMany(w http.ResponseWriter, data ...string) error {
	for _, b := range data {
		if _, err := w.Write([]byte(b)); err != nil {
			return err
		}
	}
	return nil
}

func writeJson(w http.ResponseWriter, name string, data interface{}) error {
	err := writeMany(w, `<script>window.`, name, `=`)
	if err != nil {
		return err
	}
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(`</script>`))
	return err
}

func writePage(w http.ResponseWriter, page Page) {
	_, err := w.Write([]byte(`<!doctype html><html><head><meta name="viewport" content="width=device-width, initial-scale=1.0"/>`))
	if err != nil {
		panic(err)
	}
	for name, data := range page.JsonData {
		err = writeJson(w, name, data)
		if err != nil {
			panic(err)
		}
	}

	for _, url := range page.Stylesheets {
		err = writeMany(w, `<link rel="stylesheet" href="`, url, `" />`)
		if err != nil {
			panic(err)
		}
	}

	_, err = w.Write([]byte(`</head><body><div id="app"></div>`))
	if err != nil {
		panic(err)
	}
	for _, script := range page.Scripts {
		if script.Type == "" {
			err = writeMany(w, `<script src="`, script.Url, `"></script>`)
		} else {
			err = writeMany(w, `<script src="`, script.Url, `" type="`, script.Type, `"></script>`)
		}
		if err != nil {
			panic(err)
		}
	}
	_, err = w.Write([]byte(`
		</body></html>`))
	if err != nil {
		panic(err)
	}
}
