package ui

import (
	"fmt"
	_ "image/jpeg"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	header := m.renderHeader()
	body := m.renderBody()
	footer := m.renderFooter()
	view := lipgloss.JoinVertical(lipgloss.Top, header, body, footer)
	if m.kitty && m.renderedImages && (m.clearAllImages || m.clearImagesFrames > 0) {
		if m.clearImagesFrames > 0 {
			m.clearImagesFrames--
		} else {
			m.clearAllImages = false
			m.renderedImages = false
		}
		return deleteAllImagesSeq() + view
	}
	return view
}

func (m *model) resize() {
	chatW, _, _, bodyH, _ := m.layout()
	innerMainW := chatW - 2 - 2 - scrollbarWidth
	innerMainH := bodyH - 6 - (m.toastHeight() * 2)
	if innerMainW < 10 {
		innerMainW = 10
	}
	if innerMainH < 2 {
		innerMainH = 2
	}
	m.viewport.Width = innerMainW
	m.viewport.Height = innerMainH
	inputWidth := chatW - 2 - 2 - 2
	if inputWidth < 10 {
		inputWidth = 10
	}
	m.input.Width = inputWidth
}

func (m *model) renderHeader() string {
	if m.width < 1 {
		return ""
	}
	badge := m.styles.headerBadgeOff.Render("○ CONNECTING")
	if m.roomInfo.RoomId != 0 {
		if m.roomInfo.LiveStatus == 1 {
			badge = m.styles.headerBadgeLive.Render("● LIVE")
		} else {
			badge = m.styles.headerBadgeOff.Render("○ OFFLINE")
		}
	}
	title := strings.TrimSpace(m.roomInfo.Title)
	if title == "" {
		title = "bilibili live"
	}
	left := badge + m.styles.headerTitle.Render(title)
	right := ""
	if m.roomID != 0 {
		right = m.styles.headerMeta.Render(fmt.Sprintf("room %d", m.roomID))
	}
	return alignBar(left, right, m.width, m.styles.headerBar)
}

func (m *model) renderBody() string {
	chatW, sideW, _, bodyH, _ := m.layout()
	chatParts := make([]string, 0, 4)
	toast := m.renderToast(chatW - 4)
	if toast != "" {
		chatParts = append(chatParts, toast)
	}
	viewportPane := m.renderViewportPane()
	inputStyle := m.styles.inputBox
	if m.isInputFocused() {
		inputStyle = m.styles.inputBoxHi
	}
	inputContent := m.styles.inputInner.Width(m.input.Width).Render(m.input.View())
	inputBox := inputStyle.Render(inputContent)
	usedHeight := lipgloss.Height(viewportPane) + lipgloss.Height(inputBox)
	if toast != "" {
		usedHeight += lipgloss.Height(toast)
	}
	chatParts = append(chatParts, viewportPane)
	if fillerHeight := bodyH - usedHeight; fillerHeight > 0 {
		filler := make([]string, fillerHeight)
		chatParts = append(chatParts, strings.Join(filler, "\n"))
	}
	chatParts = append(chatParts, inputBox)
	chatContent := strings.Join(chatParts, "\n")
	chatBox := m.styles.panel.Width(chatW).Height(bodyH).Render(chatContent)
	bodyStyle := lipgloss.NewStyle().PaddingRight(bodyRightMargin)
	if m.isCompact() {
		return bodyStyle.Render(chatBox)
	}

	sideBox := m.styles.sidePanel.Width(sideW).Height(bodyH).Render(m.renderSidebar(bodyH))
	body := lipgloss.JoinHorizontal(lipgloss.Top, sideBox, strings.Repeat(" ", panelGap), chatBox)
	return bodyStyle.Render(body)
}

func (m *model) renderViewportPane() string {
	content := lipgloss.JoinHorizontal(lipgloss.Top, m.viewport.View(), m.renderViewportScrollbar())
	if m.focusMessages {
		return m.styles.messageCardHi.Render(content)
	}
	return m.styles.messageCard.Render(content)
}

func (m *model) renderViewportScrollbar() string {
	height := m.viewport.Height
	if height <= 0 {
		return ""
	}
	total := len(m.lines)
	lines := make([]string, height)
	for i := range lines {
		lines[i] = m.styles.scrollTrack.Render(" ")
	}
	if total <= height {
		return strings.Join(lines, "\n")
	}
	thumb := height * height / total
	if thumb < 1 {
		thumb = 1
	}
	if thumb > height {
		thumb = height
	}
	maxOffset := total - height
	maxThumbTop := height - thumb
	thumbTop := 0
	if maxOffset > 0 && maxThumbTop > 0 {
		thumbTop = m.viewport.YOffset * maxThumbTop / maxOffset
	}
	for i := thumbTop; i < thumbTop+thumb && i < len(lines); i++ {
		lines[i] = m.styles.scrollThumb.Render("▌")
	}
	return strings.Join(lines, "\n")
}

func (m *model) renderToast(width int) string {
	if m.toastText == "" {
		return ""
	}
	if width < 10 {
		width = 10
	}
	tag := m.styles.toastTag.Render("ENTER")
	textWidth := width - lipgloss.Width(tag) - 1
	if textWidth < 1 {
		textWidth = 1
	}
	return tag + " " + m.styles.toastText.Width(textWidth).Render(trimToWidth(m.toastText, textWidth))
}

func (m *model) renderSidebar(height int) string {
	innerW := m.sidebarWidth() - 2
	if innerW < 10 {
		innerW = 10
	}
	contentW := innerW
	lines := []string{}
	lines = append(lines, m.renderRoomsSection(contentW)...)
	lines = append(lines, "")
	if m.kitty && m.sidebarCover != "" {
		cols := contentW
		if cols < 6 {
			cols = 6
		}
		coverLines := m.placeImageLines(m.sidebarCover, cols, sidebarCoverRows)
		if len(coverLines) > 0 {
			lines = append(lines, coverLines...)
			lines = append(lines, "")
		}
	}
	lines = append(lines, m.styles.label.Render("Room Info"))
	if m.roomInfo.RoomId == 0 {
		lines = append(lines, m.styles.muted.Render("No room info"))
	} else {
		title := trimToWidth(m.roomInfo.Title, contentW)
		area := trimToWidth(strings.TrimSpace(m.roomInfo.ParentAreaName+" / "+m.roomInfo.AreaName), contentW)
		lines = append(lines, title)
		if area != "" {
			lines = append(lines, m.styles.muted.Render(area))
		}
		if m.roomInfo.Online > 0 {
			lines = append(lines, m.statLine("Heat", fmt.Sprintf("%d", m.roomInfo.Online)))
		}
		lines = append(lines, m.statLine("Follows", fmt.Sprintf("%d", m.roomInfo.Attention)))
		if m.roomInfo.Time != "" {
			lines = append(lines, m.statLine("Live", m.roomInfo.Time))
		}
	}

	totalAudience := m.audienceTotal()
	audienceUsers := m.sidebarAudienceUsers()
	postLines := []string{"", m.styles.label.Render("Access")}
	postLines = append(postLines, m.previewSideSection(m.accessLines, "No access data", sidebarEventTail)...)
	postLines = append(postLines, "", m.styles.label.Render("Gift"))
	postLines = append(postLines, m.previewSideSection(m.giftLines, "No gift data", sidebarEventTail)...)
	audienceHeight := height - len(lines) - len(postLines)
	if audienceHeight > defaultAudienceRows {
		audienceHeight = defaultAudienceRows
	}
	lines = append(lines, m.renderAudienceSection(contentW, audienceHeight, totalAudience, audienceUsers)...)
	lines = append(lines, postLines...)

	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m *model) renderRoomsSection(width int) []string {
	lines := []string{m.styles.label.Render("Rooms")}
	lines = append(lines, m.renderRoomsLines(max(1, width-2))...)
	return m.renderSectionBox(lines, width, 0, m.focusPicker && !m.focusAudience)
}

func (m *model) renderSectionBox(lines []string, width int, innerHeight int, focused bool) []string {
	innerW := width - 2
	if innerW < 1 {
		innerW = 1
	}
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed = append(trimmed, trimToWidth(line, innerW))
	}
	if innerHeight > 0 {
		if len(trimmed) > innerHeight {
			trimmed = trimmed[:innerHeight]
		}
		for len(trimmed) < innerHeight {
			trimmed = append(trimmed, "")
		}
	}
	style := m.styles.sideSection
	if focused {
		style = m.styles.sideSectionHi
	}
	rendered := style.Width(innerW).Render(strings.Join(trimmed, "\n"))
	return strings.Split(rendered, "\n")
}

func (m *model) renderRoomsBlock(width int) []string {
	label := m.styles.label.Render("Rooms")
	label = label + " " + m.styles.muted.Render("Tab")
	lines := []string{label}
	lines = append(lines, m.renderRoomsLines(width)...)
	return lines
}

func (m *model) renderRoomsLines(width int) []string {
	if len(m.recentRooms) == 0 {
		return []string{m.styles.muted.Render("No rooms")}
	}
	lines := make([]string, 0, len(m.recentRooms))
	for i, r := range m.recentRooms {
		selected := m.focusPicker && i == m.pickerIdx
		lines = append(lines, m.renderRoomLine(r, width, selected, false))
	}
	return lines
}

func (m *model) renderRoomLine(r recentRoom, width int, selected bool, showAvatar bool) string {
	avatar := ""
	avatarWidth := 0
	if m.kitty && showAvatar {
		if key := m.roomAvatarKey[r.ID]; key != "" {
			avatar = m.placeImage(key, 2, 1, 1) + strings.Repeat(" ", 2)
			avatarWidth = 4
		}
	}
	name := r.Name
	if name == "" {
		name = "Unknown"
	}
	nameStyle := m.styles.roomName
	statusPrefix := m.styles.roomOffStatus.Render("○ ")
	if r.LiveStatus == 1 {
		statusPrefix = m.styles.roomLiveStatus.Render("● ")
	} else {
		nameStyle = m.styles.roomOffline
	}
	prefix := " "
	if selected {
		prefix = "›"
	}
	text := fmt.Sprintf("%s%s%s (%d)", prefix, statusPrefix, nameStyle.Render(name), r.ID)
	if r.ID == m.roomID {
		text += m.styles.label.Render(" ·you")
	}
	if width > 0 {
		textWidth := width - avatarWidth
		if textWidth < 1 {
			textWidth = 1
		}
		text = trimToWidth(text, textWidth)
	}
	line := avatar + text
	if selected {
		if avatar != "" {
			line = avatar + m.styles.roomSelected.Render(text)
		} else {
			line = m.styles.roomSelected.Render(text)
		}
	}
	return line
}

func (m *model) previewSideSection(lines []string, empty string, limit int) []string {
	if len(lines) == 0 {
		return []string{m.styles.muted.Render(empty)}
	}
	if limit <= 0 || len(lines) <= limit {
		return append([]string{}, lines...)
	}
	return append([]string{}, lines[len(lines)-limit:]...)
}

func (m *model) renderDivider(width int) string {
	if width < 1 {
		return ""
	}
	return m.styles.muted.Render(strings.Repeat("─", width))
}

func (m *model) statLine(label string, value string) string {
	if label == "" {
		return value
	}
	tag := m.styles.muted.Render(fmt.Sprintf("%-7s", label))
	return tag + " " + m.styles.content.Render(value)
}

func (m *model) renderFooter() string {
	statusLabel := iconStatus + " CONNECT"
	statusStyle := m.styles.statusSegOff
	if m.roomInfo.RoomId != 0 {
		if m.roomInfo.LiveStatus == 1 {
			statusLabel = iconStatus + " LIVE"
			statusStyle = m.styles.statusSegLive
		} else {
			statusLabel = iconStatus + " OFFLINE"
			statusStyle = m.styles.statusSegOff
		}
	}
	statusSeg := statusStyle.Render(statusLabel)
	viewerLabel := iconViewers + " AUD --"
	viewerStyle := m.styles.statusSegMuted
	if totalAudience := m.audienceTotal(); totalAudience > 0 {
		viewerLabel = fmt.Sprintf("%s AUD %d", iconViewers, totalAudience)
		viewerStyle = m.styles.statusSeg
	}
	viewerSeg := viewerStyle.Render(viewerLabel)
	liveDurationSeg := ""
	if m.roomInfo.LiveStatus == 1 && strings.TrimSpace(m.roomInfo.Time) != "" {
		liveDurationSeg = m.styles.statusSeg.Render(fmt.Sprintf("%s UP %s", iconClock, m.roomInfo.Time))
	}
	apiSeg := m.styles.statusSegMuted.Render(iconFPS + " API --")
	if m.roomInfo.APILatencyMs > 0 {
		apiSeg = m.styles.statusSeg.Render(fmt.Sprintf("%s API %dms", iconFPS, m.roomInfo.APILatencyMs))
	}
	wsSeg := m.styles.statusSegMuted.Render(iconStatus + " WS --")
	if m.roomInfo.WSLatencyMs > 0 {
		wsSeg = m.styles.statusSeg.Render(fmt.Sprintf("%s WS %dms", iconStatus, m.roomInfo.WSLatencyMs))
	}
	joiner := m.styles.statusBar.Render(" · ")
	leftSegs := []string{statusSeg}
	if liveDurationSeg != "" {
		leftSegs = append(leftSegs, liveDurationSeg)
	}
	leftSegs = append(leftSegs, viewerSeg, apiSeg, wsSeg)
	statusLeft := strings.Join(leftSegs, joiner)

	focusSeg := m.styles.statusSegFocus.Render(m.footerFocusLabel())
	nowSeg := m.styles.statusSegMuted.Render(iconClock + " " + time.Now().Format("15:04"))
	rightSegs := []string{focusSeg, nowSeg}
	if m.searchMode {
		query := strings.TrimSpace(m.input.Value())
		if query == "" {
			rightSegs = append(rightSegs, m.styles.statusSeg.Render("SEARCH"))
		} else {
			rightSegs = append(rightSegs, m.styles.statusSeg.Render("SEARCH "+trimToWidth(query, 18)))
		}
	} else if m.searchQuery != "" {
		count := len(m.searchMatches)
		if count > 0 {
			rightSegs = append(rightSegs, m.styles.statusSeg.Render(fmt.Sprintf("FIND %s %d/%d", trimToWidth(m.searchQuery, 14), m.searchIdx+1, count)))
		} else {
			rightSegs = append(rightSegs, m.styles.statusSegWarn.Render("FIND 0"))
		}
	}
	compactHints := m.width < 90
	if m.deleteConfirmID != 0 {
		rightSegs = append(rightSegs, m.styles.statusSegWarn.Render(fmt.Sprintf("DEL %d Y/N", m.deleteConfirmID)))
		rightSegs = append(rightSegs, m.styles.statusKey.Render("y Confirm"), m.styles.statusKey.Render("n Cancel"))
	} else if m.selectMode {
		mode := "SELECT"
		if m.selectRange {
			mode = "VISUAL"
		}
		rightSegs = append(rightSegs, m.styles.statusSeg.Render(mode))
		if compactHints {
			rightSegs = append(rightSegs, m.styles.statusKey.Render("j/k Move"), m.styles.statusKey.Render("y Copy"), m.styles.statusKey.Render("esc Exit"))
		} else {
			rightSegs = append(rightSegs,
				m.styles.statusKey.Render("j/k Move"),
				m.styles.statusKey.Render("y Copy"),
				m.styles.statusKey.Render("p Paste"),
				m.styles.statusKey.Render("V Range"),
				m.styles.statusKey.Render("esc Exit"),
				m.styles.statusKey.Render("Ctrl+C Quit"),
			)
		}
	} else if m.focusPicker {
		if compactHints {
			if m.focusAudience {
				rightSegs = append(rightSegs, m.styles.statusKey.Render("Tab Msg"), m.styles.statusKey.Render("j/k Aud"), m.styles.statusKey.Render("g/G"))
			} else {
				rightSegs = append(rightSegs, m.styles.statusKey.Render("Tab Aud"), m.styles.statusKey.Render("j/k Room"), m.styles.statusKey.Render("Enter Switch"))
			}
		} else {
			if m.focusAudience {
				rightSegs = append(rightSegs,
					m.styles.statusKey.Render("Tab Messages"),
					m.styles.statusKey.Render("j/k Aud"),
					m.styles.statusKey.Render("g/G Top/End"),
					m.styles.statusKey.Render("Ctrl+u/d Page"),
					m.styles.statusKey.Render("Tab Input"),
					m.styles.statusKey.Render("Ctrl+C Quit"),
				)
			} else {
				rightSegs = append(rightSegs,
					m.styles.statusKey.Render("Tab Audience"),
					m.styles.statusKey.Render("j/k Room"),
					m.styles.statusKey.Render("g/G Top/End"),
					m.styles.statusKey.Render("Enter Switch"),
					m.styles.statusKey.Render("d Delete"),
					m.styles.statusKey.Render("Tab Input"),
					m.styles.statusKey.Render("Ctrl+C Quit"),
				)
			}
		}
	} else if m.focusMessages {
		if compactHints {
			rightSegs = append(rightSegs, m.styles.statusKey.Render("Tab Input"), m.styles.statusKey.Render("j/k Msg"), m.styles.statusKey.Render("g/G"))
		} else {
			rightSegs = append(rightSegs,
				m.styles.statusKey.Render("Tab Input"),
				m.styles.statusKey.Render("j/k Msg"),
				m.styles.statusKey.Render("g/G Top/End"),
				m.styles.statusKey.Render("Ctrl+u/d Page"),
				m.styles.statusKey.Render("v Select"),
				m.styles.statusKey.Render("Ctrl+C Quit"),
			)
		}
	} else if m.searchMode {
		if compactHints {
			rightSegs = append(rightSegs, m.styles.statusKey.Render("Enter Find"), m.styles.statusKey.Render("esc Cancel"))
		} else {
			rightSegs = append(rightSegs,
				m.styles.statusKey.Render("Enter Find"),
				m.styles.statusKey.Render("esc Cancel"),
				m.styles.statusKey.Render("Ctrl+C Quit"),
			)
		}
	} else {
		if compactHints {
			rightSegs = append(rightSegs, m.styles.statusKey.Render("Tab Rooms"), m.styles.statusKey.Render("Enter Send"), m.styles.statusKey.Render("Ctrl+C Quit"))
		} else {
			rightSegs = append(rightSegs,
				m.styles.statusKey.Render("Tab Rooms"),
				m.styles.statusKey.Render("Tab Audience"),
				m.styles.statusKey.Render("Tab Messages"),
				m.styles.statusKey.Render("/ Search"),
				m.styles.statusKey.Render(": Cmd"),
				m.styles.statusKey.Render("n/N Next"),
				m.styles.statusKey.Render("Enter Send"),
				m.styles.statusKey.Render("Ctrl+C Quit"),
			)
		}
	}
	statusRight := strings.Join(rightSegs, joiner)
	status := alignBar(statusLeft, statusRight, m.width, m.styles.statusBar)
	return status
}

func (m *model) footerFocusLabel() string {
	switch {
	case m.deleteConfirmID != 0:
		return "CONFIRM"
	case m.selectMode && m.selectRange:
		return "VISUAL"
	case m.selectMode:
		return "SELECT"
	case m.searchMode:
		return "SEARCH"
	case m.focusPicker && m.focusAudience:
		return "AUDIENCE"
	case m.focusPicker:
		return "ROOMS"
	case m.focusMessages:
		return "MESSAGES"
	default:
		return "INPUT"
	}
}

func trimToWidth(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	return ansi.Truncate(s, w, "")
}

func alignBar(left, right string, width int, padStyle lipgloss.Style) string {
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	if leftW+rightW >= width {
		return trimToWidth(left, width)
	}
	gap := width - leftW - rightW
	return left + padStyle.Render(strings.Repeat(" ", gap)) + right
}

func (m model) headerHeight() int {
	return 1
}

func (m *model) isCompact() bool {
	return m.width < compactMinWidth || m.height < compactMinHeight
}

func (m *model) layout() (chatW int, sideW int, eventsW int, bodyH int, rightW int) {
	bodyH = m.height - m.headerHeight() - footerHeight
	bodyH -= panelBorderSize
	if bodyH < 5 {
		bodyH = 5
	}
	if m.isCompact() {
		return max(10, m.width-bodyRightMargin-panelBorderSize), 0, 0, bodyH, 0
	}
	sideW = m.sidebarWidth()
	chatW = m.width - sideW - panelGap - bodyRightMargin - panelBorderSize*2
	if chatW < 20 {
		sideW = 0
		chatW = m.width - bodyRightMargin - panelBorderSize
	}
	if chatW < 10 {
		chatW = 10
	}
	return chatW, sideW, eventsW, bodyH, 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
