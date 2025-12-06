package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSlug(t *testing.T) {
	assert.Equal(t, "godspeed-you-black-emperor", GeneratePersonalProjectSlug("Godspeed You! Black Emperor"))
	assert.Equal(t, "", GeneratePersonalProjectSlug("!@#$%^&"))
	assert.Equal(t, "foo-bar", GeneratePersonalProjectSlug("-- Foo Bar --"))
	assert.Equal(t, "foo-bar", GeneratePersonalProjectSlug("--foo-bar"))
	assert.Equal(t, "foo-bar", GeneratePersonalProjectSlug("foo--bar"))
	assert.Equal(t, "foo-bar", GeneratePersonalProjectSlug("foo-bar--"))
	assert.Equal(t, "foo-bar", GeneratePersonalProjectSlug("  Foo  Bar  "))
	assert.Equal(t, "20-000-leagues-under-the-sea", GeneratePersonalProjectSlug("20,000 Leagues Under the Sea"))
}
