package getter

import (
	"net/http"
	"strings"

	"github.com/noahlias/bili-live-tui/internal/config"

	myhttp "github.com/BYT0723/go-tools/http"
	"github.com/tidwall/gjson"
)

const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"

func buildBaseHeader() http.Header {
	header := http.Header{}
	header.Set("User-Agent", defaultUserAgent)
	header.Set("Referer", "https://www.bilibili.com/")

	cookie := buildCookieHeader()
	if cookie != "" {
		header.Set("Cookie", cookie)
	}
	return header
}

func buildCookieHeader() string {
	cookies := parseCookies(config.Config.Cookie)
	if cookies["buvid3"] == "" || cookies["buvid4"] == "" {
		b3, b4 := fetchBuvid34()
		if cookies["buvid3"] == "" && b3 != "" {
			cookies["buvid3"] = b3
		}
		if cookies["buvid4"] == "" && b4 != "" {
			cookies["buvid4"] = b4
		}
	}
	return formatCookies(cookies)
}

func parseCookies(cookieStr string) map[string]string {
	out := make(map[string]string)
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		out[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return out
}

func formatCookies(cookies map[string]string) string {
	if len(cookies) == 0 {
		return ""
	}
	parts := make([]string, 0, len(cookies))
	for k, v := range cookies {
		if k == "" || v == "" {
			continue
		}
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "; ")
}

func fetchBuvid34() (string, string) {
	header := http.Header{}
	header.Set("User-Agent", defaultUserAgent)
	header.Set("Referer", "https://www.bilibili.com/")
	_, body, err := myhttp.Get("https://api.bilibili.com/x/frontend/finger/spi", header, nil)
	if err != nil {
		return "", ""
	}
	b3 := gjson.GetBytes(body, "data.b_3").String()
	b4 := gjson.GetBytes(body, "data.b_4").String()
	return b3, b4
}
