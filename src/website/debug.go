package website

func StyleTest(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("style_test.html", nil, c.Perf)
	return res
}
