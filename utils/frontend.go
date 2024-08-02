package utils

import (
	"encoding/json"
	"net/http"
)

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

func ServeAuthFrontend(w http.ResponseWriter, frontendData AuthFrontendData) error {
	_, err := w.Write([]byte(`<!doctype html><html><head><script>window.auth_info=`))
	if err != nil {
		return err
	}

	err = json.NewEncoder(w).Encode(frontendData)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(`</script></head><body><div id='app'></div>
			<script type="module" src="http://localhost:5173/@vite/client"></script>
			<script type="module" src="http://localhost:5173/src/main.js"></script>
		</body></html>`))
	if err != nil {
		return err
	}

	return nil
}
