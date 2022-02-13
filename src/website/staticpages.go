package website

import "git.handmade.network/hmn/hmn/src/templates"

func Manifesto(c *RequestContext) ResponseData {
	baseData := getBaseDataAutocrumb(c, "Handmade Manifesto")
	baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    "Modern computer hardware is amazing. Manufacturers have orchestrated billions of pieces of silicon into terrifyingly complex and efficient structures…",
	})

	var res ResponseData
	res.MustWriteTemplate("manifesto.html", baseData, c.Perf)
	return res
}

func About(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("about.html", getBaseDataAutocrumb(c, "About"), c.Perf)
	return res
}

func CommunicationGuidelines(c *RequestContext) ResponseData {
	baseData := getBaseDataAutocrumb(c, "Communication Guidelines")
	baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    "The Handmade community strives to create an environment conducive to innovation, education, and constructive discussion. These are the principles we expect members to respect.",
	})

	var res ResponseData
	res.MustWriteTemplate("communication_guidelines.html", baseData, c.Perf)
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
