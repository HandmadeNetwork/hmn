package hmnurl

import (
	"net/url"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/oops"
)

const StaticPath = "/public"
const StaticThemePath = "/public/themes"

type Q struct {
	Name  string
	Value string
}

var baseUrlParsed url.URL

func init() {
	parsed, err := url.Parse(config.Config.BaseUrl)
	if err != nil {
		panic(oops.New(err, "could not parse base URL"))
	}

	baseUrlParsed = *parsed
}

func Url(path string, query []Q) string {
	return ProjectUrl(path, query, "")
}

func ProjectUrl(path string, query []Q, subdomain string) string {
	host := baseUrlParsed.Host
	if len(subdomain) > 0 {
		host = subdomain + "." + host
	}

	url := url.URL{
		Scheme:   baseUrlParsed.Scheme,
		Host:     host,
		Path:     trim(path),
		RawQuery: encodeQuery(query),
	}

	return url.String()
}

func StaticUrl(path string, query []Q) string {
	return Url(StaticPath+"/"+trim(path), query)
}

func StaticThemeUrl(path string, theme string, query []Q) string {
	return Url(StaticThemePath+"/"+theme+"/"+trim(path), query)
}

func trim(path string) string {
	if path[0] == '/' {
		return path[1:]
	}
	return path
}

func encodeQuery(query []Q) string {
	result := url.Values{}
	for _, q := range query {
		result.Set(q.Name, q.Value)
	}
	return result.Encode()
}
