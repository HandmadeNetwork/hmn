package website

func LibraryNotPortedYet(c *RequestContext) ResponseData {
	baseData := getBaseData(c, "Library", nil)

	var res ResponseData
	res.MustWriteTemplate("library_not_ported_yet.html", baseData, c.Perf)
	return res
}
