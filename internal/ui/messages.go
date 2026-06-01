package ui

import (
	"fmt"
	_ "image/jpeg"
	"strings"
	"time"

	"github.com/noahlias/bili-live-tui/internal/config"
	"github.com/noahlias/bili-live-tui/internal/getter"
	"github.com/noahlias/bili-live-tui/internal/sender"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func sendCmd(roomID int64, msg string, busChan chan getter.DanmuMsg) tea.Cmd {
	return func() tea.Msg {
		sender.SendMsg(roomID, msg, busChan)
		return nil
	}
}

func (m *model) appendMessage(msg getter.DanmuMsg) {
	if strings.TrimSpace(msg.Content) == "" {
		return
	}
	if m.isAccessMessage(msg) {
		m.appendAccess(msg)
		return
	}
	if m.isGiftMessage(msg) {
		m.appendGift(msg)
		return
	}
	m.msgs = append(m.msgs, msg)
	if len(m.msgs) > maxMessages {
		m.msgs = m.msgs[len(m.msgs)-maxMessages:]
	}
	if m.searchQuery != "" {
		m.refreshSearchMatches(false)
	}
	m.renderMessages()
}

func (m *model) entryToastText(msg getter.DanmuMsg) string {
	if msg.Type != "INTERACT_WORD" {
		return ""
	}
	if msg.Author == "" || strings.TrimSpace(msg.Content) != "进入了房间" {
		return ""
	}
	return fmt.Sprintf("%s %s", msg.Author, msg.Content)
}

func (m *model) showToast(text string) tea.Cmd {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	m.toastText = text
	m.toastSeq++
	seq := m.toastSeq
	m.resize()
	m.renderMessages()
	return tea.Tick(entryToastTTL, func(time.Time) tea.Msg {
		return toastExpiredMsg{seq: seq}
	})
}

func (m *model) clearToast() {
	if m.toastText == "" {
		return
	}
	m.toastText = ""
	m.resize()
	m.renderMessages()
}

func (m *model) toastHeight() int {
	if m.toastText == "" {
		return 0
	}
	return 1
}

func (m *model) enterSearchMode() {
	m.searchMode = true
	m.searchDraft = m.input.Value()
	m.input.SetValue(m.searchQuery)
	m.input.Prompt = "/ "
	m.input.Placeholder = "Search messages"
	m.input.Focus()
}

func (m *model) exitSearchMode() {
	m.searchMode = false
	m.input.SetValue(m.searchDraft)
	m.searchDraft = ""
	m.input.Prompt = "> "
	m.input.Placeholder = "Send a message"
	if !m.focusPicker {
		m.input.Focus()
	}
}

func (m *model) applySearch(val string) {
	m.searchQuery = strings.TrimSpace(val)
	m.refreshSearchMatches(true)
	m.renderMessages()
}

func (m *model) clearSearch() {
	m.searchQuery = ""
	m.searchMatches = nil
	m.searchIdx = 0
	m.renderMessages()
}

func (m *model) refreshSearchMatches(reset bool) {
	if m.searchQuery == "" {
		m.searchMatches = nil
		m.searchIdx = 0
		return
	}
	current := -1
	if !reset && m.hasSearchMatch() && m.searchIdx >= 0 && m.searchIdx < len(m.searchMatches) {
		current = m.searchMatches[m.searchIdx]
	}
	query := strings.ToLower(m.searchQuery)
	matches := make([]int, 0)
	for i, msg := range m.msgs {
		haystack := strings.ToLower(strings.TrimSpace(msg.Author + " " + msg.Content))
		if strings.Contains(haystack, query) {
			matches = append(matches, i)
		}
	}
	m.searchMatches = matches
	if len(matches) == 0 {
		m.searchIdx = 0
		return
	}
	if reset {
		m.searchIdx = 0
		return
	}
	for i, idx := range matches {
		if idx == current {
			m.searchIdx = i
			return
		}
	}
	if m.searchIdx >= len(matches) {
		m.searchIdx = len(matches) - 1
	}
}

func (m *model) jumpSearch(delta int) {
	if !m.hasSearchMatch() {
		return
	}
	m.searchIdx += delta
	if m.searchIdx < 0 {
		m.searchIdx = len(m.searchMatches) - 1
	}
	if m.searchIdx >= len(m.searchMatches) {
		m.searchIdx = 0
	}
	m.renderMessages()
}

func (m *model) hasSearchMatch() bool {
	return m.searchQuery != "" && len(m.searchMatches) > 0
}

func (m *model) isSearchCurrent(idx int) bool {
	if !m.hasSearchMatch() || m.searchIdx < 0 || m.searchIdx >= len(m.searchMatches) {
		return false
	}
	return m.searchMatches[m.searchIdx] == idx
}

func (m *model) renderMessages() {
	lines := make([]string, 0, len(m.msgs)*2)
	if m.selectMode && len(m.msgs) > 0 && m.selectIdx >= len(m.msgs) {
		m.selectIdx = len(m.msgs) - 1
	}
	if len(m.msgs) > 0 {
		m.msgLineIdx = make([]int, len(m.msgs))
	} else {
		m.msgLineIdx = nil
	}
	var prev getter.DanmuMsg
	nextInlineAvatars := m.shouldRenderInlineChatAvatars()
	if m.inlineChatAvatars && !nextInlineAvatars {
		m.clearImages = true
	}
	m.inlineChatAvatars = nextInlineAvatars
	for i, msg := range m.msgs {
		m.msgLineIdx[i] = len(lines)
		selected := m.isMessageSelected(i) || (!m.selectMode && m.isSearchCurrent(i))
		lines = append(lines, m.formatMessage(prev, msg, selected)...)
		prev = msg
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	target := len(lines)
	if m.viewport.Height > target {
		target = m.viewport.Height
	}
	if m.lastRenderLines > target {
		target = m.lastRenderLines
	}
	if target > len(lines) {
		pad := strings.Repeat(" ", max(1, m.viewport.Width))
		for i := len(lines); i < target; i++ {
			lines = append(lines, pad)
		}
	}
	if m.kitty && m.renderedImages && m.clearImages {
		if len(lines) == 0 {
			lines = append(lines, deleteAllImagesSeq())
		} else {
			lines[0] = deleteAllImagesSeq() + lines[0]
		}
		m.clearImages = false
		m.renderedImages = false
	}
	m.lines = lines
	m.lastRenderLines = len(lines)
	m.viewport.SetContent(strings.Join(m.lines, "\n"))
	if m.selectMode {
		m.ensureSelectionVisible()
	} else if m.hasSearchMatch() {
		m.ensureSearchVisible()
	} else {
		m.viewport.GotoBottom()
	}
}

func (m *model) shouldRenderInlineChatAvatars() bool {
	return m.kitty && len(m.msgs) <= maxInlineAvatarMessages
}

func (m *model) isEventMessage(msg getter.DanmuMsg) bool {
	switch msg.Type {
	case "NOTICE_MSG", "INTERACT_WORD", "SEND_GIFT", "COMBO_SEND", "GUARD_BUY", "USER_TOAST_MSG":
		return true
	default:
		return strings.EqualFold(msg.Author, "system")
	}
}

func (m *model) eventTagStyle(tag string) lipgloss.Style {
	switch tag {
	case "gift":
		return m.styles.eventGift
	case "notice":
		return m.styles.label
	case "live":
		return m.styles.eventLive
	default:
		return m.styles.muted
	}
}

func (m *model) eventTag(msg getter.DanmuMsg) string {
	switch msg.Type {
	case "INTERACT_WORD":
		return "entry"
	case "SEND_GIFT", "COMBO_SEND", "GUARD_BUY", "USER_TOAST_MSG":
		return "gift"
	case "NOTICE_MSG":
		return "notice"
	default:
		if strings.EqualFold(msg.Author, "system") {
			return "live"
		}
		return "event"
	}
}

func (m *model) formatMessage(prev getter.DanmuMsg, msg getter.DanmuMsg, selected bool) []string {
	isSystem := strings.EqualFold(msg.Author, "system") || msg.Type == "NOTICE_MSG"
	isEvent := m.isEventMessage(msg)
	timeStr := msg.Time.Format("15:04")
	if config.Config.ShowTime == 0 {
		timeStr = ""
	}

	formatTime := func() string {
		if timeStr == "" {
			return ""
		}
		return m.styles.time.Render(timeStr)
	}

	avatarCols := 0
	if m.kitty {
		avatarCols = 2
	}
	avatarPrefix := ""
	avatarPad := ""
	if avatarCols > 0 && !isSystem && m.inlineChatAvatars {
		uid := msg.UID
		if uid == 0 && msg.Author != "" {
			uid = m.nameToUID[msg.Author]
		}
		avatarPrefix = m.placeImage(m.avatarKeyForUID(uid), avatarCols, 1, 1)
		avatarPad = strings.Repeat(" ", avatarCols)
		if avatarPrefix != "" {
			avatarPrefix += avatarPad
		} else {
			avatarPrefix = avatarPad
		}
	}

	if config.Config.SingleLine == 1 {
		parts := []string{}
		if timeStr != "" {
			parts = append(parts, formatTime())
		}
		if isSystem {
			parts = append(parts, m.styles.system.Render(msg.Author))
			content := msg.Content
			if selected {
				content = m.styles.messageSelected.Render(content)
			} else {
				content = m.styles.system.Render(content)
			}
			parts = append(parts, content)
			return []string{avatarPrefix + strings.Join(parts, " ")}
		}
		parts = append(parts, m.styles.name.Render(msg.Author))
		content := msg.Content
		if selected {
			content = m.styles.messageSelected.Render(content)
		} else {
			content = m.styles.content.Render(content)
		}
		parts = append(parts, content)
		return []string{avatarPrefix + strings.Join(parts, " ")}
	}

	if isEvent {
		tag := m.eventTag(msg)
		parts := []string{}
		if timeStr != "" {
			parts = append(parts, formatTime())
		}
		parts = append(parts, m.eventTagStyle(tag).Render("["+tag+"]"))
		content := msg.Content
		switch {
		case selected:
			content = m.styles.messageSelected.Render(content)
		case tag == "gift":
			content = m.styles.content.Render(content)
		default:
			content = m.styles.muted.Render(content)
		}
		parts = append(parts, content)
		return []string{avatarPrefix + strings.Join(parts, " ")}
	}

	headerNeeded := prev.Type != msg.Type || prev.Author != msg.Author
	lines := []string{}
	if headerNeeded {
		labelParts := []string{}
		if isSystem {
			tag := m.eventTag(msg)
			labelParts = append(labelParts, m.eventTagStyle(tag).Render("["+tag+"]"))
		} else {
			labelParts = append(labelParts, m.styles.name.Render(msg.Author))
		}
		if timeStr != "" {
			labelParts = append(labelParts, formatTime())
		}
		lines = append(lines, avatarPrefix+strings.Join(labelParts, " "))
	}

	content := msg.Content
	switch {
	case selected:
		content = m.styles.messageSelected.Render(content)
	case isSystem:
		content = m.styles.system.Render(content)
	default:
		content = m.styles.content.Render(content)
	}
	gutter := m.styles.msgGutter
	if selected {
		gutter = m.styles.msgGutterHi
	}
	lines = append(lines, avatarPad+gutter.Render("│ ")+content)
	return lines
}
