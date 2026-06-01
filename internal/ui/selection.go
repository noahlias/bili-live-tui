package ui

import (
	_ "image/jpeg"
	"os"
	"os/exec"
	"strings"

	"github.com/noahlias/bili-live-tui/internal/getter"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) enterSelectMode() bool {
	if len(m.msgs) == 0 {
		return false
	}
	m.selectMode = true
	m.selectIdx = len(m.msgs) - 1
	m.selectRange = false
	m.selectRangeStart = m.selectIdx
	m.selectRangeEnd = m.selectIdx
	m.renderMessages()
	return true
}

func (m *model) exitSelectMode() {
	if !m.selectMode {
		return
	}
	m.selectMode = false
	m.selectRange = false
	m.renderMessages()
}

func (m *model) selectUp() {
	if m.selectIdx > 0 {
		m.selectIdx--
	}
	m.updateSelectRange()
}

func (m *model) selectDown() {
	if m.selectIdx < len(m.msgs)-1 {
		m.selectIdx++
	}
	m.updateSelectRange()
}

func (m *model) ensureSelectionVisible() {
	if !m.selectMode || len(m.msgs) == 0 || m.selectIdx < 0 || m.selectIdx >= len(m.msgs) {
		return
	}
	if m.viewport.Height <= 0 {
		return
	}
	start := m.msgLineIdx[m.selectIdx]
	end := start
	if m.selectIdx+1 < len(m.msgLineIdx) {
		end = m.msgLineIdx[m.selectIdx+1] - 1
	} else if len(m.lines) > 0 {
		end = len(m.lines) - 1
	}
	if start < m.viewport.YOffset {
		m.viewport.YOffset = start
		return
	}
	if end >= m.viewport.YOffset+m.viewport.Height {
		offset := end - m.viewport.Height + 1
		if offset < 0 {
			offset = 0
		}
		m.viewport.YOffset = offset
	}
}

func (m *model) ensureSearchVisible() {
	if !m.hasSearchMatch() || m.viewport.Height <= 0 {
		return
	}
	idx := m.searchMatches[m.searchIdx]
	if idx < 0 || idx >= len(m.msgLineIdx) {
		return
	}
	start := m.msgLineIdx[idx]
	end := start
	if idx+1 < len(m.msgLineIdx) {
		end = m.msgLineIdx[idx+1] - 1
	} else if len(m.lines) > 0 {
		end = len(m.lines) - 1
	}
	if start < m.viewport.YOffset {
		m.viewport.YOffset = start
		return
	}
	if end >= m.viewport.YOffset+m.viewport.Height {
		offset := end - m.viewport.Height + 1
		if offset < 0 {
			offset = 0
		}
		m.viewport.YOffset = offset
	}
}

func (m *model) selectedMessage() (getter.DanmuMsg, bool) {
	if len(m.msgs) == 0 || m.selectIdx < 0 || m.selectIdx >= len(m.msgs) {
		return getter.DanmuMsg{}, false
	}
	return m.msgs[m.selectIdx], true
}

func (m *model) yankSelected() {
	text := m.selectedMessagesText()
	if text == "" {
		return
	}
	m.yankBuffer = text
	_ = writeClipboard(text)
}

func (m *model) pasteYank() {
	if m.yankBuffer == "" {
		return
	}
	m.input.SetValue(m.input.Value() + m.yankBuffer)
}

func (m *model) copyMessageText(msg getter.DanmuMsg) string {
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return ""
	}
	return content
}

func (m *model) selectedMessagesText() string {
	if len(m.msgs) == 0 {
		return ""
	}
	if m.selectRange {
		start, end := m.selectRangeBounds()
		if start < 0 || end < 0 || start >= len(m.msgs) || end >= len(m.msgs) {
			return ""
		}
		lines := make([]string, 0, end-start+1)
		for i := start; i <= end; i++ {
			text := m.copyMessageText(m.msgs[i])
			if text != "" {
				lines = append(lines, text)
			}
		}
		return strings.Join(lines, "\n")
	}
	msg, ok := m.selectedMessage()
	if !ok {
		return ""
	}
	return m.copyMessageText(msg)
}

func (m *model) toggleSelectRange() {
	m.selectRange = !m.selectRange
	m.selectRangeStart = m.selectIdx
	m.selectRangeEnd = m.selectIdx
}

func (m *model) updateSelectRange() {
	if m.selectRange {
		m.selectRangeEnd = m.selectIdx
	}
}

func (m *model) selectRangeBounds() (int, int) {
	if m.selectRangeStart <= m.selectRangeEnd {
		return m.selectRangeStart, m.selectRangeEnd
	}
	return m.selectRangeEnd, m.selectRangeStart
}

func (m *model) isMessageSelected(idx int) bool {
	if !m.selectMode {
		return false
	}
	if m.selectRange {
		start, end := m.selectRangeBounds()
		return idx >= start && idx <= end
	}
	return idx == m.selectIdx
}

func (m *model) openEditorCmd() tea.Cmd {
	if m.editorPath != "" {
		return nil
	}
	tmp, err := os.CreateTemp("", "bili-tui-*.txt")
	if err != nil {
		return nil
	}
	path := tmp.Name()
	_, _ = tmp.WriteString(m.input.Value())
	_ = tmp.Close()
	m.editorPath = path
	editor := os.Getenv("EDITOR")
	if strings.TrimSpace(editor) == "" {
		editor = "nvim"
	}
	cmd := exec.Command(editor, path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{err: err, path: path}
	})
}

func writeClipboard(text string) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
