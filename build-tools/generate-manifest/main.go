package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"go/format"
	"io"
	"os"
	"text/template"
)

//go:embed manifest.tmpl
var tmplString string

type TemplateData struct {
	PackageName  string
	VariableName string
	Entrypoint   string
	Styles       []string
}

type Manifest = map[string]ManifestEntry

type ManifestEntry struct {
	File    string   `json:"file"`
	Name    string   `json:"name"`
	Src     string   `json:"src"`
	IsEntry bool     `json:"isEntry"`
	Css     []string `json:"css"`
}

func main() {
	var inputFile string
	var outputFile string
	var name string
	var packageName string

	flag.StringVar(&inputFile, "i", "", "manifest file")
	flag.StringVar(&outputFile, "o", "", "output file")
	flag.StringVar(&name, "n", "", "variable name")
	flag.StringVar(&packageName, "p", "", "package name")
	flag.Parse()

	if inputFile == "" {
		panic("missing input file (-i)")
	}

	if name == "" {
		panic("missing name (-n)")
	}

	if packageName == "" {
		panic("missing package name (-p)")
	}

	inputJson, err := os.Open(inputFile)
	if err != nil {
		panic(err)
	}

	var manifest Manifest
	err = json.NewDecoder(inputJson).Decode(&manifest)
	if err != nil {
		panic(err)
	}

	var entryPoints []ManifestEntry

	for _, entry := range manifest {
		if entry.IsEntry {
			entryPoints = append(entryPoints, entry)
		}
	}

	if len(entryPoints) != 1 {
		panic("expected exactly one entry point in the manifest")
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

	tmpl, err := template.New("static").Parse(tmplString)
	if err != nil {
		panic(err)
	}

	var output bytes.Buffer

	err = tmpl.Execute(&output, TemplateData{
		PackageName:  packageName,
		VariableName: name,
		Entrypoint:   entryPoints[0].File,
		Styles:       entryPoints[0].Css,
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
