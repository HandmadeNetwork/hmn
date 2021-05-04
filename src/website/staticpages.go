package website

import ()

func Manifesto(c *RequestContext) ResponseData {
	var res ResponseData
	res.WriteTemplate("manifesto.html", getBaseData(c), c.Perf)
	return res
}

func About(c *RequestContext) ResponseData {
	var res ResponseData
	res.WriteTemplate("about.html", getBaseData(c), c.Perf)
	return res
}

func CodeOfConduct(c *RequestContext) ResponseData {
	var res ResponseData
	res.WriteTemplate("code_of_conduct.html", getBaseData(c), c.Perf)
	return res
}

func CommunicationGuidelines(c *RequestContext) ResponseData {
	var res ResponseData
	res.WriteTemplate("communication_guidelines.html", getBaseData(c), c.Perf)
	return res
}

func ContactPage(c *RequestContext) ResponseData {
	var res ResponseData
	res.WriteTemplate("contact.html", getBaseData(c), c.Perf)
	return res
}

func MonthlyUpdatePolicy(c *RequestContext) ResponseData {
	var res ResponseData
	res.WriteTemplate("monthly_update_policy.html", getBaseData(c), c.Perf)
	return res
}

func ProjectSubmissionGuidelines(c *RequestContext) ResponseData {
	var res ResponseData
	res.WriteTemplate("project_submission_guidelines.html", getBaseData(c), c.Perf)
	return res
}
