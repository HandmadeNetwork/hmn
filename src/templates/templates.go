package templates

import (
	"embed"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/hmnurl"
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
	"currentprojecturl": func(url string) string {
		return hmnurl.Url(url, nil) // TODO: Use project subdomain
	},
	"currentprojecturlq": func(url string, query string) string {
		absUrl := hmnurl.Url(url, nil)
		return fmt.Sprintf("%s?%s", absUrl, query) // TODO: Use project subdomain
	},
	"darken": func(hexColor string, amount float64) (string, error) {
		if len(hexColor) < 6 {
			return "", fmt.Errorf("couldn't darken invalid hex color: %v", hexColor)
		}
		return noire.NewHex(hexColor).Shade(amount).Hex(), nil
	},
	"projecturl": func(url string, proj interface{}) string {
		return hmnurl.ProjectUrl(url, nil, getProjectSubdomain(proj))
	},
	"projecturlq": func(url string, proj interface{}, query string) string {
		absUrl := hmnurl.ProjectUrl(url, nil, getProjectSubdomain(proj))
		return fmt.Sprintf("%s?%s", absUrl, query)
	},
	"query": func(args ...string) string {
		query := url.Values{}
		for i := 0; i < len(args); i += 2 {
			query.Set(args[i], args[i+1])
		}
		return query.Encode()
	},
	"static": func(filepath string) string {
		return hmnurl.StaticUrl(filepath, []hmnurl.Q{{"v", cachebust}})
	},
	"staticnobust": func(filepath string) string {
		return hmnurl.StaticUrl(filepath, nil)
	},
	"statictheme": func(theme string, filepath string) string {
		return hmnurl.StaticThemeUrl(filepath, theme, []hmnurl.Q{{"v", cachebust}})
	},
	"staticthemenobust": func(theme string, filepath string) string {
		return hmnurl.StaticThemeUrl(filepath, theme, nil)
	},
	"url": func(url string) string {
		return hmnurl.Url(url, nil)
	},
	"urlq": func(url string, query string) string {
		absUrl := hmnurl.Url(url, nil)
		return fmt.Sprintf("%s?%s", absUrl, query)
	},
}

type ErrInvalidHexColor struct {
	color string
}

func (e ErrInvalidHexColor) Error() string {
	return fmt.Sprintf("invalid hex color: %s", e.color)
}

func getProjectSubdomain(proj interface{}) string {
	subdomain := ""
	switch p := proj.(type) {
	case Project:
		subdomain = p.Subdomain
	case int:
		// TODO: Look up project from the database
	default:
		panic(fmt.Errorf("projecturl requires either a templates.Project or a project ID, got %+v", proj))
	}

	return subdomain
}
