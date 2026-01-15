package website

func HSFLanding(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("hsf_landing.html", getHSFBaseData(), c.Perf)
	return res
}

func HSFAbout(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("hsf_about.html", getHSFBaseData(), c.Perf)
	return res
}

func HSFManifesto(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("hsf_manifesto.html", getHSFBaseData(), c.Perf)
	return res
}

func HSFValues(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("hsf_values.html", getHSFBaseData(), c.Perf)
	return res
}

func HSFMembership(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("hsf_membership.html", getHSFBaseData(), c.Perf)
	return res
}

func HSFProjects(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("hsf_projects.html", getHSFBaseData(), c.Perf)
	return res
}
