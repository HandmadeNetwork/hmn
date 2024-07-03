package templates

import (
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/Masterminds/sprig"
	"github.com/google/uuid"
	"github.com/teacat/noire"
)

const (
	Dayish   = time.Hour * 24
	Weekish  = Dayish * 7
	Monthish = Dayish * 30
	Yearish  = Dayish * 365
)

//go:embed src
var embeddedTemplateFs embed.FS
var embeddedTemplates map[string]*template.Template

//go:embed src/fishbowls
var FishbowlFS embed.FS

func getTemplatesFromFS(templateFS fs.ReadDirFS) (map[string]*template.Template, map[string]error) {
	templates := make(map[string]*template.Template)
	errs := make(map[string]error)

	files := utils.Must1(templateFS.ReadDir("src"))
	for _, f := range files {
		if hasSuffix(f.Name(), ".html") {
			t := template.New(f.Name())
			t = t.Funcs(sprig.FuncMap())
			t = t.Funcs(HMNTemplateFuncs)
			t, err := t.ParseFS(templateFS,
				"src/layouts/*",
				"src/include/*",
				"src/"+f.Name(),
			)
			if err != nil {
				errs[f.Name()] = err
				continue
			}

			templates[f.Name()] = t
		} else if hasSuffix(f.Name(), ".css", ".js", ".xml") {
			t := template.New(f.Name())
			t = t.Funcs(sprig.FuncMap())
			t = t.Funcs(HMNTemplateFuncs)
			t, err := t.ParseFS(templateFS, "src/"+f.Name())
			if err != nil {
				logging.Fatal().Str("filename", f.Name()).Err(err).Msg("failed to parse template")
			}

			templates[f.Name()] = t
		}
	}

	return templates, errs
}

func Init() {
	var errs map[string]error
	type errEntry struct {
		name string
		err  error
	}

	embeddedTemplates, errs = getTemplatesFromFS(embeddedTemplateFs)
	if len(errs) > 0 {
		var errsList []errEntry
		for filename, err := range errs {
			errsList = append(errsList, errEntry{filename, err})
		}
		sort.Slice(errsList, func(i, j int) bool {
			return strings.Compare(errsList[i].name, errsList[j].name) < 0
		})
		for _, err := range errsList {
			logging.Error().Str("filename", err.name).Err(err.err).Msg("Failed to parse template")
		}
		panic("Failed to parse templates; see above")
	}
}

func GetTemplate(name string) *template.Template {
	var templates map[string]*template.Template
	if config.Config.DevConfig.LiveTemplates {
		var errs map[string]error
		templates, errs = getTemplatesFromFS(utils.DirFS("src/templates").(fs.ReadDirFS))
		if errs[name] != nil {
			panic(oops.New(errs[name], "Error in template %s", name))
		}
	} else {
		templates = embeddedTemplates
	}

	template, hasTemplate := templates[name]
	if !hasTemplate {
		panic(oops.New(nil, "Template not found: %s", name))
	}
	return template
}

func hasSuffix(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

func names(ts []*template.Template) []string {
	result := make([]string, len(ts))
	for i, t := range ts {
		result[i] = t.Name()
	}
	return result
}

//go:embed img/*
var Imgs embed.FS

func GetImg(file string) []byte {
	var imgs fs.FS
	if config.Config.DevConfig.LiveTemplates {
		imgs = utils.DirFS("src/templates/img")
	} else {
		imgs = Imgs
		file = filepath.Join("img/", file)
	}

	img, err := imgs.Open(file)
	if err != nil {
		panic(err)
	}

	return utils.Must1(io.ReadAll(img))
}

func ListImgsDir(dir string) []fs.DirEntry {
	var imgs fs.ReadDirFS
	if config.Config.DevConfig.LiveTemplates {
		imgs = utils.DirFS("src/templates/img").(fs.ReadDirFS)
	} else {
		imgs = Imgs
		dir = filepath.Join("img/", dir)
	}

	entries, err := imgs.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	return entries
}

var controlCharRegex = regexp.MustCompile(`\p{Cc}`)

var HMNTemplateFuncs = template.FuncMap{
	"add": func(a int, b ...int) int {
		for _, num := range b {
			a += num
		}
		return a
	},
	"strjoin": func(strs ...string) string {
		return strings.Join(strs, "")
	},
	"absolutedate": func(t time.Time) string {
		return t.UTC().Format("January 2, 2006, 3:04pm")
	},
	"absoluteshortdate": func(t time.Time) string {
		return t.UTC().Format("January 2, 2006")
	},
	"rfc3339": func(t time.Time) string {
		return t.UTC().Format(time.RFC3339)
	},
	"rfc1123": func(t time.Time) string {
		return t.UTC().Format(time.RFC1123)
	},
	"alpha": func(alpha float64, color noire.Color) noire.Color {
		color.Alpha = alpha
		return color
	},
	"brighten": func(amount float64, color noire.Color) noire.Color {
		return color.Tint(amount)
	},
	"color2css": func(color noire.Color) template.CSS {
		return template.CSS(color.HTML())
	},
	"csrftoken": func(s Session) template.HTML {
		return template.HTML(fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`, auth.CSRFFieldName, s.CSRFToken))
	},
	"csrftokenjs": func(s Session) template.HTML {
		return template.HTML(fmt.Sprintf(`{ "field": "%s", "token": "%s" }`, auth.CSRFFieldName, s.CSRFToken))
	},
	"darken": func(amount float64, color noire.Color) noire.Color {
		return color.Shade(amount)
	},
	"hex2color": func(hex string) (noire.Color, error) {
		if len(hex) < 6 {
			return noire.Color{}, fmt.Errorf("hex color was invalid: %v", hex)
		}
		return noire.NewHex(hex), nil
	},
	"lightness": func(lightness float64, color noire.Color) noire.Color {
		h, s, _, a := color.HSLA()
		return noire.NewHSLA(h, s, lightness*100, a)
	},
	"relativedate": func(t time.Time) string {
		// TODO: Support relative future dates

		// NOTE(asaf): Months and years aren't exactly accurate, but good enough for now I guess.
		str := func(primary int, primaryName string, secondary int, secondaryName string) string {
			result := fmt.Sprintf("%d %s", primary, primaryName)
			if primary != 1 {
				result += "s"
			}
			if secondary > 0 {
				result += fmt.Sprintf(", %d %s", secondary, secondaryName)

				if secondary != 1 {
					result += "s"
				}
			}

			return result + " ago"
		}

		delta := time.Now().Sub(t)

		if delta < time.Minute {
			return "Less than a minute ago"
		} else if delta < time.Hour {
			return str(int(delta.Minutes()), "minute", 0, "")
		} else if delta < Dayish {
			return str(int(delta/time.Hour), "hour", int((delta%time.Hour)/time.Minute), "minute")
		} else if delta < Weekish {
			return str(int(delta/Dayish), "day", int((delta%Dayish)/time.Hour), "hour")
		} else if delta < Monthish {
			return str(int(delta/Weekish), "week", int((delta%Weekish)/Dayish), "day")
		} else if delta < Yearish {
			return str(int(delta/Monthish), "month", int((delta%Monthish)/Weekish), "week")
		} else {
			return str(int(delta/Yearish), "year", int((delta%Yearish)/Monthish), "month")
		}
	},
	"svg": func(name string) template.HTML {
		contents := GetImg(fmt.Sprintf("%s.svg", name))
		return template.HTML(contents)
	},
	"static": func(filepath string) string {
		return hmnurl.BuildPublic(filepath, true)
	},
	"staticnobust": func(filepath string) string {
		return hmnurl.BuildPublic(filepath, false)
	},
	"statictheme": func(theme string, filepath string) string {
		return hmnurl.BuildTheme(filepath, theme, true)
	},
	"staticthemenobust": func(theme string, filepath string) string {
		return hmnurl.BuildTheme(filepath, theme, false)
	},
	"dataimg": func(filepath string) template.URL {
		contents := GetImg(filepath)
		mime := http.DetectContentType(contents)
		contentsBase64 := base64.StdEncoding.EncodeToString(contents)
		return template.URL(fmt.Sprintf("data:%s;base64,%s", mime, contentsBase64))
	},
	"string2uuid": func(s string) string {
		return uuid.NewSHA1(uuid.NameSpaceURL, []byte(s)).URN()
	},
	"timehtml": func(formatted string, t time.Time) template.HTML {
		iso := t.UTC().Format(time.RFC3339)
		return template.HTML(fmt.Sprintf(`<time datetime="%s">%s</time>`, iso, formatted))
	},
	"timehtmlcontent": func(t time.Time) template.HTML {
		iso := t.UTC().Format(time.RFC3339)
		return template.HTML(fmt.Sprintf(`<time data-type="content" datetime="%s"></time>`, iso))
	},
	"noescape": func(str string) template.HTML {
		return template.HTML(str)
	},
	"yesescape": func(html template.HTML) string {
		return string(html)
	},
	"filesize": func(numBytes int) string {
		scales := []string{
			" bytes",
			"kb",
			"mb",
			"gb",
		}
		num := float64(numBytes)
		scale := 0
		for num > 1024 && scale < len(scales)-1 {
			num /= 1024
			scale += 1
		}
		precision := 0
		if scale > 0 {
			precision = 2
		}
		return fmt.Sprintf("%.*f%s", precision, num, scales[scale])
	},
	"cleancontrolchars": func(str template.HTML) template.HTML {
		return template.HTML(controlCharRegex.ReplaceAllString(string(str), ""))
	},
	"trim": func(str template.HTML) template.HTML {
		return template.HTML(strings.TrimSpace(string(str)))
	},
	"lastidx": func(idx int, l int) bool {
		return idx == l-1
	},
	"isodd": func(num int) bool {
		return num%2 == 1
	},

	// NOTE(asaf): Template specific functions:
	"imageselectordata": func(name string, src *Asset, required bool) ImageSelectorData {
		return ImageSelectorData{
			Name:     name,
			Asset:    src,
			Required: required,
		}
	},

	"mediaimage":   func() TimelineItemMediaType { return TimelineItemMediaTypeImage },
	"mediavideo":   func() TimelineItemMediaType { return TimelineItemMediaTypeVideo },
	"mediaaudio":   func() TimelineItemMediaType { return TimelineItemMediaTypeAudio },
	"mediaembed":   func() TimelineItemMediaType { return TimelineItemMediaTypeEmbed },
	"mediaunknown": func() TimelineItemMediaType { return TimelineItemMediaTypeUnknown },
}
