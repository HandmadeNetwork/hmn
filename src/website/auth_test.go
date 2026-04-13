package website

import (
	"testing"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"github.com/stretchr/testify/assert"
)

func TestSafeLoginRedirectUrl(t *testing.T) {
	originalBaseURL := config.Config.BaseUrl
	config.Config.BaseUrl = "https://handmade.test"
	hmnurl.SetGlobalBaseUrl(config.Config.BaseUrl)
	t.Cleanup(func() {
		config.Config.BaseUrl = originalBaseURL
		hmnurl.SetGlobalBaseUrl(config.Config.BaseUrl)
	})

	homepage := hmnurl.BuildHomepage()
	assert.Contains(t, homepage, "handmade.test") // sanity

	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl(""))
	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl("!@#$%^&*"))
	assert.Equal(t, "//handmade.test/foo/bar", hmnurl.SafeRedirectUrl("//handmade.test/foo/bar"))
	assert.Equal(t, "http://handmade.test/foo/bar", hmnurl.SafeRedirectUrl("http://handmade.test/foo/bar"))
	assert.Equal(t, "https://handmade.test/foo/bar", hmnurl.SafeRedirectUrl("https://handmade.test/foo/bar"))
	assert.Equal(t, "//foo.handmade.test/foo/bar", hmnurl.SafeRedirectUrl("//foo.handmade.test/foo/bar"))
	assert.Equal(t, "http://foo.handmade.test/foo/bar", hmnurl.SafeRedirectUrl("http://foo.handmade.test/foo/bar"))
	assert.Equal(t, "https://foo.handmade.test/foo/bar", hmnurl.SafeRedirectUrl("https://foo.handmade.test/foo/bar"))

	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl("//other.test/foo/bar"))
	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl("http://other.test/foo/bar"))
	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl("https://other.test/foo/bar"))
	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl("//nothandmade.test/foo/bar"))
	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl("http://nothandmade.test/foo/bar"))
	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl("https://nothandmade.test/foo/bar"))
	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl("//handmade.test.malicious.website/foo/bar"))
	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl("http://handmade.test.malicious.website/foo/bar"))
	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl("https://handmade.test.malicious.website/foo/bar"))

	// Relative stuff is probably fine, but there's no reason we ever need it since we always
	// generate full URLs anyhow.
	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl("/foo/bar"))
	assert.Equal(t, homepage, hmnurl.SafeRedirectUrl("handmade.test/foo/bar"))
}
