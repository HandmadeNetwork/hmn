package website

import (
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/templates"
)

func Manifesto(c *RequestContext) ResponseData {
	type TemplateData struct {
		templates.BaseData
		ValuesUrl string
	}
	baseData := getBaseTemplateData(c, "Handmade Manifesto", nil)
	baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    "Computers are amazing. So why is software so terrible?",
	})

	var res ResponseData
	res.MustWriteTemplate("manifesto.html", TemplateData{
		BaseData:  baseData,
		ValuesUrl: hmnurl.BuildValues(),
	}, c.Perf)
	return res
}

func Values(c *RequestContext) ResponseData {
	type TemplateData struct {
		templates.BaseData
		ProjectsUrl string
	}
	baseData := getBaseTemplateData(c, "Values", nil)

	var res ResponseData
	res.MustWriteTemplate("values.html", TemplateData{
		BaseData:    baseData,
		ProjectsUrl: hmnurl.BuildProjectIndex(),
	}, c.Perf)
	return res
}

func About(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("about.html", getBaseTemplateData(c, "About", nil), c.Perf)
	return res
}

func Rules(c *RequestContext) ResponseData {
	type tmpl struct {
		templates.BaseData
		AIPolicyUrl      string
		PrivacyPolicyUrl string
	}
	var res ResponseData
	res.MustWriteTemplate("rules.html", tmpl{
		BaseData:         getBaseTemplateData(c, "Rules & Policies", nil),
		AIPolicyUrl:      hmnurl.BuildAIPolicy(),
		PrivacyPolicyUrl: hmnurl.BuildPrivacyPolicy(),
	}, c.Perf)
	return res
}

func PrivacyPolicy(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("privacy.html", getBaseTemplateData(c, "Privacy Policy", nil), c.Perf)
	return res
}

func AIPolicy(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("ai_policy.html", getBaseTemplateData(c, "AI Policy", nil), c.Perf)
	return res
}

func ContactPage(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("contact.html", getBaseTemplateData(c, "Contact Us", nil), c.Perf)
	return res
}
