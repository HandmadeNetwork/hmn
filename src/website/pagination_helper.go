package website

import (
	"strconv"

	"git.handmade.network/hmn/hmn/src/utils"
)

/*
Parses a path param as a page number, and returns the parsed result and
a value indicating whether parsing was successful.

The returned page number is always valid, even when parsing fails. If
parsing fails (ok is false), you should redirect to the returned
page number.
*/
func ParsePageNumber(
	c *RequestContext,
	paramName string,
	numPages int,
) (page int, ok bool) {
	page = 1
	if pageString, hasPage := c.PathParams[paramName]; hasPage && pageString != "" {
		if pageParsed, err := strconv.Atoi(pageString); err == nil {
			page = pageParsed
		} else {
			return 1, false
		}
	}
	if page < 1 || numPages < page {
		return utils.Clamp(1, page, numPages), false
	}

	return page, true
}
