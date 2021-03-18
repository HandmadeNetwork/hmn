package hmnurl

import (
	"net/url"

	"git.handmade.network/hmn/hmn/src/config"
)

const StaticPath = "/public"
const StaticThemePath = "/public/themes"

type Q struct {
	Name  string
	Value string
}

func Url(path string, query []Q) string {
	result := config.Config.BaseUrl + "/" + trim(path)
	if q := encodeQuery(query); q != "" {
		result += "?" + q
	}
	return result
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
