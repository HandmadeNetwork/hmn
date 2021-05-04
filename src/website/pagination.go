package website

import (
	"math"
	"strconv"
)

func getPageInfo(
	pageParam string,
	totalItems int,
	itemsPerPage int,
) (
	page int,
	totalPages int,
	ok bool,
) {
	totalPages = int(math.Ceil(float64(totalItems) / float64(itemsPerPage)))
	ok = true

	page = 1
	if pageParam != "" {
		if pageParsed, err := strconv.Atoi(pageParam); err == nil {
			page = pageParsed
		} else {
			return 0, 0, false
		}
	}
	if page < 1 || totalPages < page {
		return 0, 0, false
	}

	return
}
