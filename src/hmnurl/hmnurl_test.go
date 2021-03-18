package hmnurl

import (
	"testing"

	"git.handmade.network/hmn/hmn/src/config"
	"github.com/stretchr/testify/assert"
)

func TestUrl(t *testing.T) {
	defer func(original string) {
		config.Config.BaseUrl = original
	}(config.Config.BaseUrl)
	config.Config.BaseUrl = "http://handmade.test"

	t.Run("no query", func(t *testing.T) {
		result := Url("/test/foo", nil)
		assert.Equal(t, "http://handmade.test/test/foo", result)
	})
	t.Run("yes query", func(t *testing.T) {
		result := Url("/test/foo", []Q{{"bar", "baz"}, {"zig??", "zig & zag!!"}})
		assert.Equal(t, "http://handmade.test/test/foo?bar=baz&zig%3F%3F=zig+%26+zag%21%21", result)
	})
}
