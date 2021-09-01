package website

func Manifesto(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("manifesto.html", getBaseDataAutocrumb(c, "Manifesto"), c.Perf)
	return res
}

func About(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("about.html", getBaseDataAutocrumb(c, "About"), c.Perf)
	return res
}

func CodeOfConduct(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("code_of_conduct.html", getBaseDataAutocrumb(c, "Code of Conduct"), c.Perf)
	return res
}

func CommunicationGuidelines(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("communication_guidelines.html", getBaseDataAutocrumb(c, "Communication Guidelines"), c.Perf)
	return res
}

func ContactPage(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("contact.html", getBaseDataAutocrumb(c, "Contact Us"), c.Perf)
	return res
}

func MonthlyUpdatePolicy(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("monthly_update_policy.html", getBaseDataAutocrumb(c, "Monthly Update Policy"), c.Perf)
	return res
}

func ProjectSubmissionGuidelines(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("project_submission_guidelines.html", getBaseDataAutocrumb(c, "Project Submission Guidelines"), c.Perf)
	return res
}
