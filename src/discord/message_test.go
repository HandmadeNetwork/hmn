package discord

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMessage(t *testing.T) {
	t.Skip("this test is only for debugging")

	msg, err := GetChannelMessage(context.Background(), "404399251276169217", "764575065772916790")
	assert.Nil(t, err)
	t.Logf("%+v", msg)
}
