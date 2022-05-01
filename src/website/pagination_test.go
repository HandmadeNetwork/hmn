package website

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPageInfo(t *testing.T) {
	items := []struct {
		name                string
		pageParam           string
		totalItems, perPage int
		page, totalPages    int
		ok                  bool
	}{
		{"good, no param", "", 85, 10, 1, 9, true},
		{"good", "2", 85, 10, 2, 9, true},
		{"too big", "10", 85, 10, 0, 0, false},
		{"too small", "0", 85, 10, 0, 0, false},
		{"pizza", "pizza", 85, 10, 0, 0, false},
		{"zero items, no param", "", 0, 10, 1, 1, true}, // should go to page 1
		{"zero items, page 1", "1", 0, 10, 1, 1, true},
		{"zero items, too big", "2", 0, 10, 0, 0, false},
		{"zero items, too small", "0", 0, 10, 0, 0, false},
	}

	for _, item := range items {
		t.Run(item.name, func(t *testing.T) {
			page, totalPages, ok := getPageInfo(item.pageParam, item.totalItems, item.perPage)
			assert.Equal(t, item.page, page)
			assert.Equal(t, item.totalPages, totalPages)
			assert.Equal(t, item.ok, ok)
		})
	}
}
