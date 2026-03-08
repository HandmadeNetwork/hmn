package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimple(t *testing.T) {
	input := `
<!DOCTYPE html>
<html>

<head>
<meta charset="utf-8">
</head>

<body>
<div style="background:#fff;font-size:16px">Hello!</div>
</body>

</html>
	`
	expected := `
<!DOCTYPE html>
<html>

<head>
<meta charset="utf-8">
</head>

<body>
<div style="background:#fff;font-size:16px;">Hello!</div>
</body>

</html>
	`
	actual, err := preprocessEmailHTML([]byte(input))
	assert.Nil(t, err)
	assert.Equal(t, expected, string(actual))
}

func TestSimpleCSS(t *testing.T) {
	input := `
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
	/* Hey, comments */
	body {
		font-size: 16px; /* we love comments */
	}

	.bg-white {
		/* comments are so good */
		background: #fff;
	}
	.f5 {
		font-size: /* so good */ 16px;
	}
</style>
</head>
<body>
<div class="bg-white f5" style="font-weight:bold;text-decoration:underline">Hello!</div>
</body>
</html>
	`
	expected := `
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">

</head>
<body style="font-size:16px;">
<div style="background:#fff;font-size:16px;font-weight:bold;text-decoration:underline;">Hello!</div>
</body>
</html>
	`
	actual, err := preprocessEmailHTML([]byte(input))
	assert.Nil(t, err)
	assert.Equal(t, expected, string(actual))
}
