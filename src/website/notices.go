package website

import (
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/templates"
)

const NoticesCookieName = "hmn_notices"

func getNoticesFromCookie(c *RequestContext) []templates.Notice {
	cookie, err := c.Req.Cookie(NoticesCookieName)
	if err != nil {
		if !errors.Is(err, http.ErrNoCookie) {
			c.Logger.Warn().Err(err).Msg("failed to get notices cookie")
		}
		return nil
	}
	return deserializeNoticesFromCookie(cookie.Value)
}

func storeNoticesInCookie(c *RequestContext, res *ResponseData) {
	serialized := serializeNoticesForCookie(c, res.FutureNotices)
	if serialized != "" {
		noticesCookie := http.Cookie{
			Name:     NoticesCookieName,
			Value:    serialized,
			Path:     "/",
			Domain:   config.Config.Auth.CookieDomain,
			Expires:  time.Now().Add(time.Minute * 5),
			Secure:   config.Config.Auth.CookieSecure,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
		res.SetCookie(&noticesCookie)
	} else if !(res.StatusCode >= 300 && res.StatusCode < 400) {
		// NOTE(asaf): Don't clear on redirect
		noticesCookie := http.Cookie{
			Name:   NoticesCookieName,
			Path:   "/",
			Domain: config.Config.Auth.CookieDomain,
			MaxAge: -1,
		}
		res.SetCookie(&noticesCookie)
	}
}

func serializeNoticesForCookie(c *RequestContext, notices []templates.Notice) string {
	var builder strings.Builder
	maxSize := 1024 // NOTE(asaf): Make sure we don't use too much space for notices.
	size := 0
	for i, notice := range notices {
		sizeIncrease := len(notice.Class) + len(string(notice.Content)) + 1
		if i != 0 {
			sizeIncrease += 1
		}
		if size+sizeIncrease > maxSize {
			c.Logger.Warn().Interface("Notices", notices).Msg("Notices too big for cookie")
			break
		}

		if i != 0 {
			builder.WriteString("\t")
		}
		builder.WriteString(notice.Class)
		builder.WriteString("|")
		builder.WriteString(string(notice.Content))

		size += sizeIncrease
	}
	return builder.String()
}

func deserializeNoticesFromCookie(cookieVal string) []templates.Notice {
	var result []templates.Notice
	notices := strings.Split(cookieVal, "\t")
	for _, notice := range notices {
		parts := strings.SplitN(notice, "|", 2)
		if len(parts) == 2 {
			result = append(result, templates.Notice{
				Class:   parts[0],
				Content: template.HTML(parts[1]),
			})
		}
	}
	return result
}

func storeNoticesInCookieMiddleware(h Handler) Handler {
	return func(c *RequestContext) ResponseData {
		res := h(c)
		storeNoticesInCookie(c, &res)
		return res
	}
}
