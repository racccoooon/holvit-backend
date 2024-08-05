package main

import (
	"bytes"
	_ "embed"
	"flag"
	"github.com/google/brotli/go/cbrotli"
	"go/format"
	"io"
	"maps"
	"mime"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"text/template"
	"unicode"
	"unicode/utf8"
)

//go:embed static.tmpl
var tmplString string

type File struct {
	Name           string
	ContentType    string
	DefaultVariant *Variant
	OtherVariants  map[string]*Variant
	Variants       map[string]*Variant
}

type Variant struct {
	Id      string
	Content []byte
}

type TemplateData struct {
	BuildFlags  string
	PackageName string
	TypeName    string
	StructName  string
	Variants    []*Variant
	Files       []File
}

var nameRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

type variantGen func(content []byte) ([]byte, error)

func brotliVariant(content []byte) ([]byte, error) {
	return cbrotli.Encode(content, cbrotli.WriterOptions{
		Quality: 11,
		LGWin:   0,
	})
}

func main() {
	variantGenerators := map[string]variantGen{
		"br": brotliVariant,
	}

	var sourceDir string
	var outputFile string
	var name string
	var packageName string
	var buildFlags string

	flag.StringVar(&sourceDir, "i", "", "input directory")
	flag.StringVar(&outputFile, "o", "", "output file")
	flag.StringVar(&name, "n", "", "variable name")
	flag.StringVar(&packageName, "p", "", "package name")
	flag.StringVar(&buildFlags, "f", "", "build flags")
	flag.Parse()

	if name == "" {
		panic("variable name (-n) required")
	}

	if packageName == "" {
		panic("package name (-p) required")
	}

	var out io.Writer

	if outputFile == "" {
		out = os.Stdout
	} else {
		var err error
		out, err = os.Create(outputFile)
		if err != nil {
			panic(err)
		}
	}

	var err error

	dir, err := os.ReadDir(sourceDir)
	if err != nil {
		panic(err)
	}

	var files []File
	var allVariants []*Variant

	for _, e := range dir {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		id := nameRegex.ReplaceAllString(name, "_")
		p := filepath.Join(sourceDir, name)

		content, err := os.ReadFile(p)
		if err != nil {
			panic(err)
		}

		variants := make(map[string]*Variant)

		defaultVariant := Variant{
			Id:      id,
			Content: content,
		}
		allVariants = append(allVariants, &defaultVariant)

		for encodingName, gen := range variantGenerators {
			variantContent, err := gen(content)
			if err != nil {
				panic(err)
			}
			variant := Variant{
				Id:      id + "_" + encodingName,
				Content: variantContent,
			}
			allVariants = append(allVariants, &variant)
			variants[encodingName] = &variant
		}

		otherVariants := maps.Clone(variants)
		variants[""] = &defaultVariant

		f := File{
			Name:           "/" + name,
			ContentType:    mime.TypeByExtension(path.Ext(p)),
			Variants:       variants,
			DefaultVariant: &defaultVariant,
			OtherVariants:  otherVariants,
		}
		files = append(files, f)
	}

	tmpl, err := template.New("static").Parse(tmplString)
	if err != nil {
		panic(err)
	}

	var output bytes.Buffer

	err = tmpl.Execute(&output, TemplateData{
		BuildFlags:  buildFlags,
		PackageName: packageName,
		TypeName:    firstToLower(name) + "_s",
		StructName:  name,
		Variants:    allVariants,
		Files:       files,
	})
	if err != nil {
		panic(err)
	}

	outputFormatted, err := format.Source(output.Bytes())
	if err != nil {
		panic(err)
	}

	_, err = out.Write(outputFormatted)
	if err != nil {
		panic(err)
	}
}

// https://stackoverflow.com/a/75989905
func firstToLower(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size <= 1 {
		return s
	}
	lc := unicode.ToLower(r)
	if r == lc {
		return s
	}
	return string(lc) + s[size:]
}
