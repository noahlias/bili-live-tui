package ui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/noahlias/bili-live-tui/internal/getter"

	tea "github.com/charmbracelet/bubbletea"
)

func recentRoomsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "bili", "recent_rooms.json"), nil
}

func roomAvatarPath(roomID int64) (string, error) {
	if roomID <= 0 {
		return "", fmt.Errorf("invalid room id")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "bili", "room_avatars")
	return filepath.Join(dir, fmt.Sprintf("%d.png", roomID)), nil
}

func roomAvatarKey(roomID int64) string {
	return fmt.Sprintf("room_avatar_%d", roomID)
}

func loadRecentRooms() []recentRoom {
	path, err := recentRoomsPath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var rooms []recentRoom
	if err := json.Unmarshal(data, &rooms); err == nil && len(rooms) > 0 {
		return normalizeRecentRooms(rooms)
	}
	var ids []int64
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil
	}
	rooms = make([]recentRoom, 0, len(ids))
	for _, id := range ids {
		rooms = append(rooms, recentRoom{ID: id})
	}
	return normalizeRecentRooms(rooms)
}

func saveRecentRooms(rooms []recentRoom) {
	path, err := recentRoomsPath()
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	data, err := json.Marshal(rooms)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0644)
}

func normalizeRecentRooms(rooms []recentRoom) []recentRoom {
	out := make([]recentRoom, 0, len(rooms))
	seen := make(map[int64]bool, len(rooms))
	for _, r := range rooms {
		if r.ID <= 0 || seen[r.ID] {
			continue
		}
		seen[r.ID] = true
		out = append(out, r)
		if len(out) >= maxRecentRooms {
			break
		}
	}
	return out
}

func (m *model) loadRoomAvatar(roomID int64) bool {
	path, err := roomAvatarPath(roomID)
	if err != nil {
		return false
	}
	if _, err := os.Stat(path); err != nil {
		return false
	}
	key := roomAvatarKey(roomID)
	if _, ok := m.imageCache[key]; !ok {
		payload := base64.StdEncoding.EncodeToString([]byte(path))
		m.imageCache[key] = imageEntry{payload: payload, path: path}
	}
	m.roomAvatarKey[roomID] = key
	return true
}

func (m *model) reloadRecentRoomAvatars() {
	if !m.kitty {
		return
	}
	for _, r := range m.recentRooms {
		m.loadRoomAvatar(r.ID)
	}
}

func (m *model) enterCommandMode() {
	m.searchMode = false
	m.searchDraft = ""
	m.commandMode = true
	m.input.SetValue("")
	m.input.Prompt = ":"
	m.input.Placeholder = "room 12345"
	m.input.Focus()
}

func (m *model) exitCommandMode() {
	m.commandMode = false
	m.input.SetValue("")
	m.input.Prompt = "> "
	m.input.Placeholder = "Send a message"
	if !m.focusPicker {
		m.input.Focus()
	}
}

func (m *model) runCommand(val string) tea.Cmd {
	cmd := strings.TrimSpace(val)
	if cmd == "" {
		return nil
	}
	if id, ok := parseRoomID(cmd); ok {
		return m.switchRoom(id)
	}
	m.appendMessage(getter.DanmuMsg{
		Author:  "system",
		Content: "Unknown command",
		Type:    "NOTICE_MSG",
		Time:    time.Now(),
	})
	return nil
}

func parseRoomID(cmd string) (int64, bool) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return 0, false
	}
	if parts[0] == "room" || parts[0] == "r" {
		if len(parts) < 2 {
			return 0, false
		}
		return parseRoomIDValue(parts[1])
	}
	return parseRoomIDValue(parts[0])
}

func parseRoomIDValue(s string) (int64, bool) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func (m *model) selectedRoomID() (int64, bool) {
	if len(m.recentRooms) == 0 || m.pickerIdx < 0 || m.pickerIdx >= len(m.recentRooms) {
		return 0, false
	}
	return m.recentRooms[m.pickerIdx].ID, true
}

func (m *model) pickerUp() {
	if len(m.recentRooms) == 0 {
		return
	}
	m.pickerIdx--
	if m.pickerIdx < 0 {
		m.pickerIdx = len(m.recentRooms) - 1
	}
	m.clearDeleteConfirm()
}

func (m *model) pickerDown() {
	if len(m.recentRooms) == 0 {
		return
	}
	m.pickerIdx++
	if m.pickerIdx >= len(m.recentRooms) {
		m.pickerIdx = 0
	}
	m.clearDeleteConfirm()
}

func (m *model) addRecentRoom(id int64) {
	if id <= 0 {
		return
	}
	next := []recentRoom{{ID: id}}
	for _, r := range m.recentRooms {
		if r.ID == id {
			if next[0].Name == "" {
				next[0].Name = r.Name
			}
			if next[0].LiveStatus == 0 {
				next[0].LiveStatus = r.LiveStatus
			}
			continue
		}
		next = append(next, r)
		if len(next) >= maxRecentRooms {
			break
		}
	}
	m.recentRooms = next
	m.pickerIdx = 0
	saveRecentRooms(m.recentRooms)
}

func (m *model) deleteRecentRoom(id int64) bool {
	if id <= 0 || len(m.recentRooms) == 0 {
		return false
	}
	idx := -1
	for i, r := range m.recentRooms {
		if r.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	m.recentRooms = append(m.recentRooms[:idx], m.recentRooms[idx+1:]...)
	if m.pickerIdx > idx {
		m.pickerIdx--
	}
	if m.pickerIdx >= len(m.recentRooms) {
		m.pickerIdx = len(m.recentRooms) - 1
	}
	if m.pickerIdx < 0 {
		m.pickerIdx = 0
	}
	saveRecentRooms(m.recentRooms)
	delete(m.roomAvatarKey, id)
	delete(m.imageCache, roomAvatarKey(id))
	if m.kitty {
		m.clearAllImages = true
		m.clearImagesFrames = 2
	}
	return true
}

func (m *model) updateRecentRoomName(roomID int64, name string) bool {
	if roomID <= 0 || name == "" {
		return false
	}
	updated := false
	for i := range m.recentRooms {
		if m.recentRooms[i].ID == roomID && m.recentRooms[i].Name != name {
			m.recentRooms[i].Name = name
			updated = true
			break
		}
	}
	if updated {
		saveRecentRooms(m.recentRooms)
	}
	return updated
}

func (m *model) updateRecentRoomStatus(roomID int64, status int64) bool {
	if roomID <= 0 {
		return false
	}
	updated := false
	for i := range m.recentRooms {
		if m.recentRooms[i].ID == roomID && m.recentRooms[i].LiveStatus != status {
			m.recentRooms[i].LiveStatus = status
			updated = true
			break
		}
	}
	if updated {
		saveRecentRooms(m.recentRooms)
	}
	return updated
}

func (m *model) updateRecentRoomNameByUID(uid int64, name string) {
	if uid <= 0 || name == "" || m.roomInfo.Uid == 0 {
		return
	}
	if int64(m.roomInfo.Uid) != uid {
		return
	}
	m.updateRecentRoomName(m.roomID, name)
}

func (m *model) switchRoom(id int64) tea.Cmd {
	if id <= 0 || id == m.roomID {
		return nil
	}
	m.clearDeleteConfirm()
	if m.stopGetter != nil {
		m.stopGetter()
		m.stopGetter = nil
	}
	m.roomID = id
	m.roomInfo = getter.RoomInfo{}
	m.msgs = nil
	m.lines = nil
	m.viewport.SetContent("")
	m.viewport.GotoBottom()
	m.lastRenderLines = 0
	m.selectMode = false
	m.selectIdx = 0
	m.selectRange = false
	m.selectRangeStart = 0
	m.selectRangeEnd = 0
	m.streamerUID = 0
	m.headerAvatar = ""
	m.sidebarCover = ""
	m.pendingUID = make(map[int64]bool)
	m.avatarKeyBy = make(map[int64]string)
	m.imageCache = make(map[string]imageEntry)
	m.nameToUID = make(map[string]int64)
	m.roomAvatarKey = make(map[int64]string)
	m.activeAudience = make(map[string]getter.AudienceUser)
	m.accessLines = nil
	m.giftLines = nil
	m.searchMode = false
	m.searchQuery = ""
	m.searchMatches = nil
	m.searchIdx = 0
	m.searchDraft = ""
	m.toastText = ""
	m.toastSeq = 0
	m.reloadRecentRoomAvatars()
	m.addRecentRoom(id)
	m.exitCommandMode()
	m.clearImages = true
	m.clearAllImages = true
	m.clearImagesFrames = 2
	m.renderMessages()
	getter.ResetHistory(id)
	m.stopGetter = getter.RunWithRoom(id, m.busChan, m.roomInfoChan)
	needSummary := true
	for _, r := range m.recentRooms {
		if r.ID == id && r.Name != "" {
			needSummary = false
			break
		}
	}
	avatarLoaded := false
	if m.kitty {
		avatarLoaded = m.loadRoomAvatar(id)
	}
	if !needSummary && avatarLoaded {
		return nil
	}
	if needSummary || (m.kitty && !avatarLoaded) {
		return fetchRoomSummaryCmd(id)
	}
	return nil
}

func (m *model) clearDeleteConfirm() {
	m.deleteConfirmID = 0
}

func (m *model) startDeleteConfirm() bool {
	if id, ok := m.selectedRoomID(); ok {
		m.deleteConfirmID = id
		return true
	}
	return false
}

func (m *model) confirmDeleteRecentRoom() bool {
	if m.deleteConfirmID == 0 {
		return false
	}
	id := m.deleteConfirmID
	if selected, ok := m.selectedRoomID(); !ok || selected != id {
		m.deleteConfirmID = 0
		return false
	}
	m.deleteConfirmID = 0
	return m.deleteRecentRoom(id)
}
