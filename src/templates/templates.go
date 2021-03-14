package templates

import (
	"embed"
	"fmt"
	"html/template"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/logging"
	"github.com/Masterminds/sprig"
	"github.com/teacat/noire"
)

//go:embed src
var templateFs embed.FS
var Templates map[string]*template.Template

var cachebust string

func Init() {
	cachebust = fmt.Sprint(time.Now().Unix())

	Templates = make(map[string]*template.Template)

	files, _ := templateFs.ReadDir("src")
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".html") {
			t := template.New(f.Name())
			t = t.Funcs(sprig.FuncMap())
			t = t.Funcs(HMNTemplateFuncs)
			t, err := t.ParseFS(templateFs, "src/layouts/*.html", "src/include/*.html", "src/"+f.Name())
			if err != nil {
				logging.Fatal().Str("filename", f.Name()).Err(err).Msg("failed to parse template")
			}

			Templates[f.Name()] = t
		} else if strings.HasSuffix(f.Name(), ".css") {
			t := template.New(f.Name())
			t = t.Funcs(sprig.FuncMap())
			t = t.Funcs(HMNTemplateFuncs)
			t, err := t.ParseFS(templateFs, "src/"+f.Name())
			if err != nil {
				logging.Fatal().Str("filename", f.Name()).Err(err).Msg("failed to parse template")
			}

			Templates[f.Name()] = t
		}
	}

	for name, t := range Templates {
		fmt.Printf("%s: %v\n", name, names(t.Templates()))
	}
}

func names(ts []*template.Template) []string {
	result := make([]string, len(ts))
	for i, t := range ts {
		result[i] = t.Name()
	}
	return result
}

var HMNTemplateFuncs = template.FuncMap{
	"brighten": func(hexColor string, amount float64) (string, error) {
		if len(hexColor) < 6 {
			return "", fmt.Errorf("couldn't brighten invalid hex color: %v", hexColor)
		}
		return noire.NewHex(hexColor).Tint(amount).Hex(), nil
	},
	"cachebust": func() string {
		return cachebust
	},
	"darken": func(hexColor string, amount float64) (string, error) {
		if len(hexColor) < 6 {
			return "", fmt.Errorf("couldn't darken invalid hex color: %v", hexColor)
		}
		return noire.NewHex(hexColor).Shade(amount).Hex(), nil
	},
	// TODO: Actually put paths in here, duh
	"static": func(filepath string) string {
		return fmt.Sprintf("A static file at %v, busted with %v", filepath, cachebust)
	},
	"staticnobust": func(filepath string) string {
		return fmt.Sprintf("A static file at %v", filepath)
	},
	"statictheme": func(theme string, filepath string) string {
		return fmt.Sprintf("A static file for the current theme at %v, busted with %v", filepath, cachebust)
	},
	"staticthemenobust": func(theme string, filepath string) string {
		return fmt.Sprintf("A static file for the current theme at %v", filepath)
	},
	"url": func(url string) string {
		return "/" + url
	},
}

type ErrInvalidHexColor struct {
	color string
}

func (e ErrInvalidHexColor) Error() string {
	return fmt.Sprintf("invalid hex color: %s", e.color)
}
