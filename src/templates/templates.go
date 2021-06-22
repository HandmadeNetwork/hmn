package templates

import (
	"embed"
	"fmt"
	"html/template"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"github.com/Masterminds/sprig"
	"github.com/teacat/noire"
)

const (
	Dayish   = time.Hour * 24
	Weekish  = Dayish * 7
	Monthish = Dayish * 30
	Yearish  = Dayish * 365
)

//go:embed src
var templateFs embed.FS
var Templates map[string]*template.Template

func Init() {
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
		} else if strings.HasSuffix(f.Name(), ".xml") {
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
	"add": func(a int, b ...int) int {
		for _, num := range b {
			a += num
		}
		return a
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
	"timehtml": func(formatted string, t time.Time) template.HTML {
		iso := t.UTC().Format(time.RFC3339)
		return template.HTML(fmt.Sprintf(`<time datetime="%s">%s</time>`, iso, formatted))
	},
	"noescape": func(str string) template.HTML {
		return template.HTML(str)
	},

	// NOTE(asaf): Template specific functions:
	"projectcarddata": func(project Project, classes string) ProjectCardData {
		return ProjectCardData{
			Project: &project,
			Classes: classes,
		}
	},

	"timelinepostitem": func(item TimelineItem) bool {
		if item.Type == TimelineTypeForumThread ||
			item.Type == TimelineTypeForumReply ||
			item.Type == TimelineTypeBlogPost ||
			item.Type == TimelineTypeBlogComment ||
			item.Type == TimelineTypeWikiCreate ||
			item.Type == TimelineTypeWikiEdit ||
			item.Type == TimelineTypeWikiTalk ||
			item.Type == TimelineTypeLibraryComment {

			return true
		}
		return false
	},
	"timelinesnippetitem": func(item TimelineItem) bool {
		if item.Type == TimelineTypeSnippetImage ||
			item.Type == TimelineTypeSnippetVideo ||
			item.Type == TimelineTypeSnippetAudio ||
			item.Type == TimelineTypeSnippetYoutube {

			return true
		}
		return false
	},
	"snippetvideo": func(snippet TimelineItem) bool {
		return snippet.Type == TimelineTypeSnippetVideo
	},
	"snippetaudio": func(snippet TimelineItem) bool {
		return snippet.Type == TimelineTypeSnippetAudio
	},
	"snippetimage": func(snippet TimelineItem) bool {
		return snippet.Type == TimelineTypeSnippetImage
	},
	"snippetyoutube": func(snippet TimelineItem) bool {
		return snippet.Type == TimelineTypeSnippetYoutube
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
