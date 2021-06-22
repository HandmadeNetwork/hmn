package website

import ()

func Manifesto(c *RequestContext) ResponseData {
	var res ResponseData
	err := res.WriteTemplate("manifesto.html", getBaseData(c), c.Perf)
	if err != nil {
		panic(err)
	}
	return res
}

func About(c *RequestContext) ResponseData {
	var res ResponseData
	err := res.WriteTemplate("about.html", getBaseData(c), c.Perf)
	if err != nil {
		panic(err)
	}
	return res
}

func CodeOfConduct(c *RequestContext) ResponseData {
	var res ResponseData
	err := res.WriteTemplate("code_of_conduct.html", getBaseData(c), c.Perf)
	if err != nil {
		panic(err)
	}
	return res
}

func CommunicationGuidelines(c *RequestContext) ResponseData {
	var res ResponseData
	err := res.WriteTemplate("communication_guidelines.html", getBaseData(c), c.Perf)
	if err != nil {
		panic(err)
	}
	return res
}

func ContactPage(c *RequestContext) ResponseData {
	var res ResponseData
	err := res.WriteTemplate("contact.html", getBaseData(c), c.Perf)
	if err != nil {
		panic(err)
	}
	return res
}

func MonthlyUpdatePolicy(c *RequestContext) ResponseData {
	var res ResponseData
	err := res.WriteTemplate("monthly_update_policy.html", getBaseData(c), c.Perf)
	if err != nil {
		panic(err)
	}
	return res
}

func ProjectSubmissionGuidelines(c *RequestContext) ResponseData {
	var res ResponseData
	err := res.WriteTemplate("project_submission_guidelines.html", getBaseData(c), c.Perf)
	if err != nil {
		panic(err)
	}
	return res
}
