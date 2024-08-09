package handlers

import (
	"encoding/json"
	"net/http"
)

func AdminFrontend(writer http.ResponseWriter, request *http.Request) {
	writePage(w)
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
