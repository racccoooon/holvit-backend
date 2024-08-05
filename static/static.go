package static

import (
	"errors"
	"github.com/timewasted/go-accept-headers"
	"holvit/config"
	"holvit/httpErrors"
	"net/http"
	"path"
	"strconv"
	"strings"
)

type Manifest struct {
	JsEntrypoint string
	Stylesheets  []string
}

type FileSystem interface {
	Get(name string, compression string) *File
}

type File struct {
	Content         []byte
	ContentType     string
	ContentEncoding string
}

type handler struct {
	roots []FileSystem
}

func StaticServer(roots ...FileSystem) http.Handler {
	return &handler{
		roots: roots,
	}
}

func handleStatic(w http.ResponseWriter, r *http.Request, roots []FileSystem) error {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}
	filename := path.Clean(upath)

	compression, err := accept.Negotiate(r.Header.Get("Accept-Encoding"), "br")
	if err != nil {
		return err
	}
	var file *File

	for _, root := range roots {
		file = root.Get(filename, compression)
		if file != nil {
			break
		}
	}
	if file == nil {
		return httpErrors.NotFound()
	}

	if file.ContentEncoding != "" {
		w.Header().Set("Content-Encoding", file.ContentEncoding)
	}
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(file.Content)), 10))

	_, err = w.Write(file.Content)
	if err != nil {
		return err
	}
	return nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := handleStatic(w, r, h.roots)
	if err != nil {
		var httpError *httpErrors.HttpError
		switch {
		case errors.As(err, &httpError):
			var httpErr *httpErrors.HttpError
			errors.As(err, &httpErr)
			http.Error(w, httpErr.Message(), httpErr.Status())
			break
		default:
			msg := "An internal server error occurred"

			if config.C.IsDevelopment() {
				msg = err.Error()
			}

			http.Error(w, msg, http.StatusInternalServerError)
			break
		}
	}
}
