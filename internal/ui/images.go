package ui

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/noahlias/bili-live-tui/internal/getter"

	tea "github.com/charmbracelet/bubbletea"
)

func supportsKittyGraphics() bool {
	if os.Getenv("BILI_TUI_IMAGES") == "1" {
		return true
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	if os.Getenv("WEZTERM_PANE") != "" {
		return true
	}
	return os.Getenv("GHOSTTY_RESOURCES_DIR") != ""
}

func fetchUserInfoCmd(uid int64) tea.Cmd {
	return func() tea.Msg {
		info, err := getter.GetUserInfo(uid)
		if err != nil {
			return userInfoMsg{uid: uid, err: err}
		}
		return userInfoMsg{
			uid:       uid,
			face:      info.Face,
			topPhoto:  info.TopPhoto,
			liveCover: info.LiveCover,
			name:      info.Name,
		}
	}
}

func fetchRoomSummaryCmd(roomID int64) tea.Cmd {
	return func() tea.Msg {
		summary, err := getter.GetRoomSummary(roomID)
		if err != nil {
			return roomSummaryMsg{roomID: roomID, err: err}
		}
		info, err := getter.GetUserInfo(summary.UID)
		if err != nil {
			return roomSummaryMsg{roomID: roomID, err: err}
		}
		return roomSummaryMsg{roomID: roomID, name: info.Name, face: info.Face, live: summary.LiveStatus}
	}
}

func fetchRoomStatusCmd(roomID int64) tea.Cmd {
	return func() tea.Msg {
		summary, err := getter.GetRoomSummary(roomID)
		if err != nil {
			return roomStatusMsg{roomID: roomID, err: err}
		}
		return roomStatusMsg{roomID: roomID, live: summary.LiveStatus}
	}
}

func refreshRoomsTickCmd() tea.Cmd {
	return tea.Tick(10*time.Second, func(time.Time) tea.Msg {
		return roomsRefreshMsg{}
	})
}

func fetchRoomAvatarCmd(roomID int64, url string) tea.Cmd {
	return func() tea.Msg {
		data, err := fetchImageBytes(url)
		if err != nil {
			return imageMsg{}
		}
		path, err := roomAvatarPath(roomID)
		if err != nil {
			return imageMsg{}
		}
		key := roomAvatarKey(roomID)
		return imageMsg{key: key, data: data, path: path}
	}
}

func fetchImageCmd(key, url string) tea.Cmd {
	return func() tea.Msg {
		data, err := fetchImageBytes(url)
		if err != nil {
			return imageMsg{key: key}
		}
		return imageMsg{key: key, data: data}
	}
}

func fetchImageBytes(url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("empty url")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image fetch status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return toPNG(raw)
}

func toPNG(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func imageKey(url string) string {
	sum := sha1.Sum([]byte(url))
	return "img_" + hex.EncodeToString(sum[:])
}

func (m model) avatarKeyForUID(uid int64) string {
	return m.avatarKeyBy[uid]
}

func (m *model) storeImage(key string, data []byte) bool {
	if key == "" || len(data) == 0 {
		return false
	}
	if _, ok := m.imageCache[key]; ok {
		return false
	}
	dir := filepath.Join(os.TempDir(), "bili-live-tui")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}
	path := filepath.Join(dir, key+".png")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return false
	}
	payload := base64.StdEncoding.EncodeToString([]byte(path))
	m.imageCache[key] = imageEntry{payload: payload, path: path}
	return true
}

func (m *model) storeImageAtPath(key string, path string, data []byte) bool {
	if key == "" || path == "" || len(data) == 0 {
		return false
	}
	if _, ok := m.imageCache[key]; ok {
		return false
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return false
	}
	payload := base64.StdEncoding.EncodeToString([]byte(path))
	m.imageCache[key] = imageEntry{payload: payload, path: path}
	return true
}

func (m *model) placeImage(key string, cols, rows, z int) string {
	if !m.kitty || key == "" || cols <= 0 || rows <= 0 {
		return ""
	}
	entry, ok := m.imageCache[key]
	if !ok || entry.payload == "" {
		return ""
	}
	m.renderedImages = true
	return wrapForTmux(fmt.Sprintf("\x1b_Ga=T,q=1,f=100,t=f,C=1,c=%d,r=%d,z=%d;%s\x1b\\", cols, rows, z, entry.payload))
}

func (m *model) placeImageLines(key string, cols, rows int) []string {
	if !m.kitty || key == "" || cols <= 0 || rows <= 0 {
		return nil
	}
	img := m.placeImage(key, cols, rows, 1)
	if img == "" {
		return nil
	}
	lines := []string{img}
	for i := 1; i < rows; i++ {
		lines = append(lines, "")
	}
	return lines
}

func deleteAllImagesSeq() string {
	return wrapForTmux("\x1b_Ga=d,d=A,q=1\x1b\\")
}

func wrapForTmux(seq string) string {
	if os.Getenv("TMUX") == "" {
		return seq
	}
	escaped := strings.ReplaceAll(seq, "\x1b", "\x1b\x1b")
	return "\x1bPtmux;" + escaped + "\x1b\\"
}

func clearScreenSeq() string {
	return "\x1b[2J\x1b[H"
}

func pickRoomCover(info getter.RoomInfo) (string, string) {
	candidates := []string{info.UserCover, info.Background, info.Keyframe}
	for _, url := range candidates {
		if url == "" {
			continue
		}
		return imageKey(url), url
	}
	return "", ""
}
