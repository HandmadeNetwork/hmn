package website

func Manifesto(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("manifesto.html", getBaseData(c), c.Perf)
	return res
}

func About(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("about.html", getBaseData(c), c.Perf)
	return res
}

func CodeOfConduct(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("code_of_conduct.html", getBaseData(c), c.Perf)
	return res
}

func CommunicationGuidelines(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("communication_guidelines.html", getBaseData(c), c.Perf)
	return res
}

func ContactPage(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("contact.html", getBaseData(c), c.Perf)
	return res
}

func MonthlyUpdatePolicy(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("monthly_update_policy.html", getBaseData(c), c.Perf)
	return res
}

func ProjectSubmissionGuidelines(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("project_submission_guidelines.html", getBaseData(c), c.Perf)
	return res
}
