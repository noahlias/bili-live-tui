package getter

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	myhttp "github.com/BYT0723/go-tools/http"
	"github.com/tidwall/gjson"
)

var mixinKeyEncTab = []int{
	46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35,
	27, 43, 5, 49, 33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13,
	37, 48, 7, 16, 24, 55, 40, 61, 26, 17, 0, 1, 60, 51, 30, 4,
	22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11, 36, 20, 34, 44, 52,
}

type wbiKeys struct {
	imgKey    string
	subKey    string
	mixinKey  string
	updatedAt time.Time
	mu        sync.Mutex
}

var wbiCache wbiKeys

func updateWbiFromNav(body []byte) {
	imgURL := gjson.GetBytes(body, "data.wbi_img.img_url").String()
	subURL := gjson.GetBytes(body, "data.wbi_img.sub_url").String()
	if imgURL == "" || subURL == "" {
		return
	}
	imgKey := wbiKeyFromURL(imgURL)
	subKey := wbiKeyFromURL(subURL)
	if imgKey == "" || subKey == "" {
		return
	}
	wbiCache.setKeys(imgKey, subKey)
}

func signWbiParams(params map[string]string, header http.Header) (string, error) {
	if err := wbiCache.ensure(header); err != nil {
		return "", err
	}
	return wbiCache.sign(params), nil
}

func (w *wbiKeys) setKeys(imgKey, subKey string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.imgKey = imgKey
	w.subKey = subKey
	w.mixinKey = buildMixinKey(imgKey + subKey)
	w.updatedAt = time.Now()
}

func (w *wbiKeys) ensure(header http.Header) error {
	w.mu.Lock()
	if w.mixinKey != "" && time.Since(w.updatedAt) < time.Hour {
		w.mu.Unlock()
		return nil
	}
	w.mu.Unlock()

	_, body, err := myhttp.Get("https://api.bilibili.com/x/web-interface/nav", header, nil)
	if err != nil {
		return err
	}
	imgURL := gjson.GetBytes(body, "data.wbi_img.img_url").String()
	subURL := gjson.GetBytes(body, "data.wbi_img.sub_url").String()
	if imgURL == "" || subURL == "" {
		return fmt.Errorf("wbi img keys not found")
	}
	imgKey := wbiKeyFromURL(imgURL)
	subKey := wbiKeyFromURL(subURL)
	if imgKey == "" || subKey == "" {
		return fmt.Errorf("wbi keys invalid")
	}
	w.setKeys(imgKey, subKey)
	return nil
}

func (w *wbiKeys) sign(params map[string]string) string {
	p := make(map[string]string, len(params)+2)
	for k, v := range params {
		p[k] = v
	}
	p["wts"] = strconv.FormatInt(time.Now().Unix(), 10)

	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := sanitizeWbiValue(p[k])
		parts = append(parts, encodeURIComponent(k)+"="+encodeURIComponent(v))
	}
	query := strings.Join(parts, "&")
	sum := md5.Sum([]byte(query + w.mixinKey))
	wrid := hex.EncodeToString(sum[:])
	return query + "&w_rid=" + wrid
}

func buildMixinKey(s string) string {
	var b strings.Builder
	for _, idx := range mixinKeyEncTab {
		if idx >= 0 && idx < len(s) {
			b.WriteByte(s[idx])
		}
	}
	mixin := b.String()
	if len(mixin) > 32 {
		return mixin[:32]
	}
	return mixin
}

func sanitizeWbiValue(v string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '!', '\'', '(', ')', '*':
			return -1
		default:
			return r
		}
	}, v)
}

func encodeURIComponent(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
			continue
		}
		b.WriteString(fmt.Sprintf("%%%02X", c))
	}
	return b.String()
}

func pathBase(urlStr string) string {
	if idx := strings.LastIndex(urlStr, "/"); idx >= 0 && idx+1 < len(urlStr) {
		return urlStr[idx+1:]
	}
	return urlStr
}

func wbiKeyFromURL(urlStr string) string {
	base := pathBase(urlStr)
	if dot := strings.Index(base, "."); dot > 0 {
		return base[:dot]
	}
	return base
}
