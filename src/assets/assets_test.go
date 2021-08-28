package assets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeFilename(t *testing.T) {
	assert.Equal(t, "cool filename.txt.wow", SanitizeFilename("cool filename.txt.wow"))
	assert.Equal(t, " hi doggy ", SanitizeFilename("😎 hi doggy 🐶"))
	assert.Equal(t, "newlinesaretotallylegal", SanitizeFilename("newlines\naretotallylegal"))
}
