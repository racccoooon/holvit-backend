package services

import (
	"encoding/json"
	"holvit/config"
	"holvit/services/generated"
	"net/http"
)

type FrontendService interface {
	WriteAuthFrontend(w http.ResponseWriter, frontendData AuthFrontendData) error
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
			scripts = append(scripts, Script{Url: "/static/auth.js"})
			styles = append(styles, "/static/auth.css")
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

func (f *frontendServiceImpl) WriteAuthFrontend(w http.ResponseWriter, frontendData AuthFrontendData) error {
	page := f.authPage
	page.Title = "TODO"
	page.JsonData = map[string]interface{}{
		"auth_info": frontendData,
	}
	return writePage(w, page)
}

type AuthFrontendUser struct {
	Name string `json:"name"`
}

type AuthFrontendScope struct {
	Required    bool   `json:"required"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

type AuthFrontendDataAuthorize struct {
	ClientName string              `json:"client_name"`
	User       AuthFrontendUser    `json:"user"`
	Scopes     []AuthFrontendScope `json:"scopes"`
	Token      string              `json:"token"`
	GrantUrl   string              `json:"grant_url"`
	RefuseUrl  string              `json:"refuse_url"`
	LogoutUrl  string              `json:"logout_url"`
}

type AuthFrontendDataAuthenticate struct {
	ClientName    string `json:"client_name"`
	Token         string `json:"token"`
	UseRememberMe bool   `json:"use_remember_me"`
	RegisterUrl   string `json:"register_url"`
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

func writePage(w http.ResponseWriter, page Page) error {
	_, err := w.Write([]byte(`<!doctype html><html><head><meta name="viewport" content="width=device-width, initial-scale=1.0"/>`))
	if err != nil {
		return err
	}
	for name, data := range page.JsonData {
		err = writeJson(w, name, data)
		if err != nil {
			return err
		}
	}

	for _, url := range page.Stylesheets {
		err = writeMany(w, `<link rel="stylesheet" href="`, url, `" />`)
		if err != nil {
			return err
		}
	}

	_, err = w.Write([]byte(`</head><body><div id="app"></div>`))
	if err != nil {
		return err
	}
	for _, script := range page.Scripts {
		if script.Type == "" {
			err = writeMany(w, `<script src="`, script.Url, `"></script>`)
		} else {
			err = writeMany(w, `<script src="`, script.Url, `" type="`, script.Type, `"></script>`)
		}
		if err != nil {
			return err
		}
	}
	_, err = w.Write([]byte(`
		</body></html>`))

	return err
}
