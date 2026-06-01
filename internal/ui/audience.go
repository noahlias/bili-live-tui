package ui

import (
	"fmt"
	_ "image/jpeg"
	"sort"
	"strings"

	"github.com/noahlias/bili-live-tui/internal/config"
	"github.com/noahlias/bili-live-tui/internal/getter"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) renderAudienceSection(width int, height int, totalAudience int64, users []getter.AudienceUser) []string {
	if height < 5 {
		height = 5
	}
	innerH := height - 2
	if innerH < 3 {
		innerH = 3
	}
	lines := []string{m.styles.label.Render("Audience")}
	if m.roomInfo.RoomId == 0 {
		lines = append(lines, m.styles.muted.Render("Online  --"))
	} else {
		lines = append(lines, m.statLine("Online", fmt.Sprintf("%d", totalAudience)))
	}

	listHeight := innerH - len(lines)
	if listHeight < 1 {
		listHeight = 1
	}
	if totalAudience <= 0 || len(users) == 0 {
		m.sidebarOffset = 0
		m.sidebarMaxOffset = 0
		lines = append(lines, m.styles.muted.Render("No audience data"))
		return m.renderSectionBox(lines, width, innerH, m.focusPicker && m.focusAudience)
	}

	baseListW := width - 2
	if baseListW < 8 {
		baseListW = 8
	}
	listW := baseListW
	if len(users) > listHeight && sidebarScrollW > 0 {
		listW = width - 2 - sidebarScrollW - 1
		if listW < 8 {
			listW = 8
		}
	}
	audienceLines := make([]string, 0, len(users))
	for i, u := range users {
		audienceLines = append(audienceLines, m.renderAudienceLine(i+1, u, listW))
	}
	visible := m.truncateAudienceLines(audienceLines, listHeight)
	if len(users) > listHeight && sidebarScrollW > 0 {
		scroll := m.renderAudienceScrollbar(listHeight)
		listBlock := lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(visible, "\n"), " ", scroll)
		lines = append(lines, strings.Split(listBlock, "\n")...)
	} else {
		lines = append(lines, visible...)
	}
	return m.renderSectionBox(lines, width, innerH, m.focusPicker && m.focusAudience)
}

func (m *model) sidebarAudienceUsers() []getter.AudienceUser {
	users := append([]getter.AudienceUser{}, m.roomInfo.AudienceUsers...)
	if len(users) > maxSideLines {
		users = users[:maxSideLines]
	}
	sort.SliceStable(users, func(i, j int) bool {
		left := users[i]
		right := users[j]
		if left.Rank > 0 && right.Rank > 0 && left.Rank != right.Rank {
			return left.Rank < right.Rank
		}
		if left.Rank > 0 != (right.Rank > 0) {
			return left.Rank > 0
		}
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.UID < right.UID
	})
	return users
}

func (m model) renderAudienceLine(index int, user getter.AudienceUser, width int) string {
	name := user.Name
	if name == "" {
		name = fmt.Sprintf("uid:%d", user.UID)
	}
	prefix := ""
	if index > 0 {
		prefix = fmt.Sprintf("%d ", index)
	}
	meta := compactAudienceMeta(user)
	score := "--"
	if user.Score > 0 {
		score = fmt.Sprintf("%d", user.Score)
	}
	textWidth := width - lipgloss.Width(score) - 2 - lipgloss.Width(prefix)
	if textWidth < 6 {
		textWidth = 6
	}
	nameWidth := textWidth
	if meta != "" {
		minNameWidth := 8
		metaWidth := lipgloss.Width(meta) + 1
		if textWidth-metaWidth < minNameWidth {
			allowedMeta := textWidth - minNameWidth - 1
			if allowedMeta <= 0 {
				meta = ""
			} else {
				meta = trimToWidth(meta, allowedMeta)
			}
		}
		if meta != "" {
			nameWidth = textWidth - lipgloss.Width(meta) - 1
		}
	}
	if nameWidth < 6 {
		nameWidth = 6
	}
	left := m.styles.muted.Render(prefix) + m.styles.rankName.Render(trimToWidth(name, nameWidth))
	if meta != "" {
		left += " " + m.styles.muted.Render(meta)
	}
	right := m.styles.content.Render(score)
	return left + strings.Repeat(" ", max(1, width-lipgloss.Width(left)-lipgloss.Width(right))) + right
}

func compactAudienceMeta(user getter.AudienceUser) string {
	parts := make([]string, 0, 3)
	if user.MedalLevel > 0 {
		label := compactMedalLabel(user.MedalName)
		parts = append(parts, fmt.Sprintf("%s%d", label, user.MedalLevel))
	}
	if user.GuardLevel > 0 {
		parts = append(parts, fmt.Sprintf("G%d", user.GuardLevel))
	}
	if user.WealthLevel > 0 {
		parts = append(parts, fmt.Sprintf("W%d", user.WealthLevel))
	}
	return strings.Join(parts, " ")
}

func compactMedalLabel(name string) string {
	if strings.TrimSpace(name) == "" {
		return "M"
	}
	runes := []rune(name)
	if len(runes) > 2 {
		runes = runes[:2]
	}
	return string(runes)
}

func (m *model) audienceTotal() int64 {
	usersCount := int64(len(m.sidebarAudienceUsers()))
	if m.roomInfo.OnlineRankTotal > usersCount {
		return m.roomInfo.OnlineRankTotal
	}
	if m.roomInfo.Online <= 0 {
		return m.roomInfo.OnlineRankTotal
	}
	return usersCount
}

func (m *model) renderAudienceBlock(lines []string, width int, height int) []string {
	contentW := width - sidebarScrollW - 1
	if contentW < 8 {
		contentW = 8
	}
	visible := m.truncateAudienceLines(lines, height)
	content := strings.Join(visible, "\n")
	if sidebarScrollW <= 0 {
		return visible
	}
	scroll := m.renderAudienceScrollbar(height)
	block := lipgloss.JoinHorizontal(lipgloss.Top, content, " ", scroll)
	return strings.Split(block, "\n")
}

func audienceUserMapKey(user getter.AudienceUser) string {
	if user.UID > 0 {
		return fmt.Sprintf("uid:%d", user.UID)
	}
	return "name:" + user.Name
}

func mergeAudienceUser(current getter.AudienceUser, incoming getter.AudienceUser) getter.AudienceUser {
	if current.Name == "" {
		current.Name = incoming.Name
	}
	if current.Face == "" {
		current.Face = incoming.Face
	}
	if current.Rank == 0 || (incoming.Rank > 0 && incoming.Rank < current.Rank) {
		current.Rank = incoming.Rank
	}
	if incoming.Score > current.Score {
		current.Score = incoming.Score
	}
	if current.MedalName == "" {
		current.MedalName = incoming.MedalName
	}
	if incoming.MedalLevel > current.MedalLevel {
		current.MedalLevel = incoming.MedalLevel
	}
	if incoming.GuardLevel > current.GuardLevel {
		current.GuardLevel = incoming.GuardLevel
	}
	if incoming.WealthLevel > current.WealthLevel {
		current.WealthLevel = incoming.WealthLevel
	}
	current.Sources = mergeAudienceSources(current.Sources, incoming.Sources)
	return current
}

func mergeAudienceSources(left []string, right []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(left)+len(right))
	for _, source := range append(left, right...) {
		if source == "" || seen[source] {
			continue
		}
		seen[source] = true
		out = append(out, source)
	}
	return out
}

func (m *model) truncateAudienceLines(lines []string, height int) []string {
	if height <= 0 {
		m.sidebarOffset = 0
		m.sidebarMaxOffset = 0
		return nil
	}
	if len(lines) <= height {
		m.sidebarOffset = 0
		m.sidebarMaxOffset = 0
		out := append([]string{}, lines...)
		for len(out) < height {
			out = append(out, "")
		}
		return out
	}
	maxOffset := len(lines) - height
	if m.sidebarMaxOffset > 0 && m.sidebarOffset == m.sidebarMaxOffset && maxOffset > m.sidebarMaxOffset {
		m.sidebarOffset = maxOffset
	}
	if m.sidebarOffset < 0 {
		m.sidebarOffset = 0
	}
	if m.sidebarOffset > maxOffset {
		m.sidebarOffset = maxOffset
	}
	m.sidebarMaxOffset = maxOffset
	start := m.sidebarOffset
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}
	out := append([]string{}, lines[start:end]...)
	for len(out) < height {
		out = append(out, "")
	}
	return out
}

func (m *model) sidebarHeight() int {
	if m.isCompact() {
		return 0
	}
	_, sideW, _, bodyH, _ := m.layout()
	if sideW == 0 {
		return 0
	}
	return bodyH
}

func (m *model) scrollAudience(key string) {
	if m.sidebarMaxOffset <= 0 {
		m.sidebarOffset = 0
		return
	}
	height := m.sidebarHeight()
	step := height / 2
	if step < 1 {
		step = 1
	}
	switch key {
	case "k":
		m.sidebarOffset -= paneScrollStep
	case "j":
		m.sidebarOffset += paneScrollStep
	case "ctrl+u":
		m.sidebarOffset -= step
	case "ctrl+d":
		m.sidebarOffset += step
	case "g":
		m.sidebarOffset = 0
	case "G":
		m.sidebarOffset = m.sidebarMaxOffset
	case "pgup":
		m.sidebarOffset -= step
	case "pgdown":
		m.sidebarOffset += step
	case "home":
		m.sidebarOffset = 0
	case "end":
		m.sidebarOffset = m.sidebarMaxOffset
	}
	if m.sidebarOffset < 0 {
		m.sidebarOffset = 0
	}
	if m.sidebarOffset > m.sidebarMaxOffset {
		m.sidebarOffset = m.sidebarMaxOffset
	}
}

func (m *model) scrollAudienceStep(step int) {
	if m.sidebarMaxOffset <= 0 {
		m.sidebarOffset = 0
		return
	}
	m.sidebarOffset += step
	if m.sidebarOffset < 0 {
		m.sidebarOffset = 0
	}
	if m.sidebarOffset > m.sidebarMaxOffset {
		m.sidebarOffset = m.sidebarMaxOffset
	}
}

func (m *model) renderAudienceScrollbar(height int) string {
	if height <= 0 {
		return ""
	}
	lines := make([]string, height)
	for i := range lines {
		lines[i] = m.styles.scrollTrack.Render("│")
	}
	if m.sidebarMaxOffset <= 0 {
		return strings.Join(lines, "\n")
	}
	total := m.sidebarMaxOffset + height
	thumb := height * height / total
	if thumb < 1 {
		thumb = 1
	}
	if thumb > height {
		thumb = height
	}
	maxThumbTop := height - thumb
	thumbTop := 0
	if maxThumbTop > 0 {
		thumbTop = m.sidebarOffset * maxThumbTop / m.sidebarMaxOffset
	}
	for i := thumbTop; i < thumbTop+thumb && i < len(lines); i++ {
		lines[i] = m.styles.scrollThumb.Render("█")
	}
	return strings.Join(lines, "\n")
}

func (m model) sidebarWidth() int {
	if m.width < 100 {
		return sidebarMinW
	}
	return sidebarWideW
}

func (m model) isAccessMessage(msg getter.DanmuMsg) bool {
	return msg.Type == "INTERACT_WORD"
}

func (m model) isGiftMessage(msg getter.DanmuMsg) bool {
	switch msg.Type {
	case "SEND_GIFT", "COMBO_SEND", "GUARD_BUY", "USER_TOAST_MSG":
		return true
	default:
		return false
	}
}

func (m *model) appendAccess(msg getter.DanmuMsg) {
	line := m.formatSideMessage(msg)
	if line == "" {
		return
	}
	m.accessLines = append(m.accessLines, line)
	if len(m.accessLines) > maxSideLines {
		m.accessLines = m.accessLines[len(m.accessLines)-maxSideLines:]
	}
}

func (m *model) appendGift(msg getter.DanmuMsg) {
	line := m.formatSideMessage(msg)
	if line == "" {
		return
	}
	m.giftLines = append(m.giftLines, line)
	if len(m.giftLines) > maxSideLines {
		m.giftLines = m.giftLines[len(m.giftLines)-maxSideLines:]
	}
}

func (m *model) recordAudienceActivity(msg getter.DanmuMsg) {
	if strings.TrimSpace(msg.Author) == "" || strings.EqualFold(msg.Author, "system") {
		return
	}
	user := getter.AudienceUser{
		UID:  msg.UID,
		Name: msg.Author,
	}
	switch msg.Type {
	case "DANMU_MSG":
		user.Sources = []string{"danmu"}
	case "INTERACT_WORD":
		user.Sources = []string{"entry"}
	case "SEND_GIFT", "COMBO_SEND", "GUARD_BUY", "USER_TOAST_MSG":
		user.Sources = []string{"gift"}
	default:
		user.Sources = []string{"active"}
	}
	key := audienceUserMapKey(user)
	if existing, ok := m.activeAudience[key]; ok {
		m.activeAudience[key] = mergeAudienceUser(existing, user)
		return
	}
	m.activeAudience[key] = user
}

func (m model) formatSideMessage(msg getter.DanmuMsg) string {
	if strings.TrimSpace(msg.Content) == "" {
		return ""
	}
	timeStr := msg.Time.Format("15:04")
	if config.Config.ShowTime == 0 {
		timeStr = ""
	}
	parts := []string{}
	if timeStr != "" {
		parts = append(parts, m.styles.time.Render(timeStr))
	}
	if msg.Author != "" && msg.Author != "system" {
		parts = append(parts, m.styles.name.Render(msg.Author))
	}
	parts = append(parts, m.styles.content.Render(msg.Content))
	return strings.Join(parts, " ")
}
