//+build ignore

// reference from: https://github.com/koddr/example-embed-static-files-go/blob/master/internal/box/generator.go

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"yunion.io/x/log"
)

const (
	blobFileName string = "zz_generated.blob.go"
	embedFolder  string = "../../static"
)

// Define vars for build template
var conv = map[string]interface{}{
	"conv":      fmtByteSlice,
	"constName": constName,
}
var tmpl = template.Must(template.New("").Funcs(conv).Parse(`package embed

// Code generated by go generate; DO NOT EDIT.

const (
	{{- range $name, $file := . }}
		{{ constName $name }} = "{{ $name }}"
	{{- end }}
)

func init() {
	{{- range $name, $file := . }}
		box.Add("{{ $name }}", []byte{ {{ conv $file }} })
	{{- end }}
}`))

func fmtByteSlice(s []byte) string {
	builder := strings.Builder{}

	for _, v := range s {
		builder.WriteString(fmt.Sprintf("%d,", int(v)))
	}

	return builder.String()
}

func constName(keyPath string) string {
	keyPath = strings.TrimPrefix(keyPath, "/")
	keyPath = strings.ReplaceAll(keyPath, "-", "_")
	keyPath = strings.ReplaceAll(keyPath, ".", "_")
	keyPath = strings.ReplaceAll(keyPath, "/", "_")
	keyPath = strings.ToUpper(keyPath)
	return keyPath
}

func main() {
	// Checking directory with files
	if _, err := os.Stat(embedFolder); os.IsNotExist(err) {
		log.Fatalf("Static directory does not exists!")
	}

	// Create map for filenames
	configs := make(map[string][]byte)

	// Walking through embed directory
	err := filepath.Walk(embedFolder, func(path string, info os.FileInfo, err error) error {
		relativePath := filepath.ToSlash(strings.TrimPrefix(path, embedFolder))

		if info.IsDir() {
			// Skip directories
			log.Warningf("%s is a directory, skipping...", path)
			return nil
		} else {
			// If element is a simple file, embed
			b, err := ioutil.ReadFile(path)
			if err != nil {
				log.Errorf("Error read %s: %v", path, err)
				return err
			}

			// Add file name to map
			log.Infof("Add %s => %s", path, relativePath)
			configs[relativePath] = b
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error walking through embed directory: %v", err)
	}

	// Delete blob file
	os.Remove(blobFileName)

	// Create blob file
	f, err := os.Create(blobFileName)
	if err != nil {
		log.Fatalf("Error creating blob file: %v", err)
	}
	defer f.Close()

	// Create buffer
	builder := &bytes.Buffer{}

	// Execute template
	if err := tmpl.Execute(builder, configs); err != nil {
		log.Fatalf("Error executing template: %v", err)
	}

	// Formatting generated code
	data, err := format.Source(builder.Bytes())
	if err != nil {
		log.Fatalf("Error formatting generated code: %v, code: \n%s", err, builder.Bytes())
	}

	// Writing blob file
	if err := ioutil.WriteFile(blobFileName, data, os.ModePerm); err != nil {
		log.Fatalf("Error writing blob file: %v", err)
	}
}
