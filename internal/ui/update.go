package ui

import (
	_ "image/jpeg"
	"os"
	"strings"
	"time"

	"github.com/noahlias/bili-live-tui/internal/config"
	"github.com/noahlias/bili-live-tui/internal/getter"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		m.clearAllImages = true
		m.renderMessages()
		return m, nil
	case tea.KeyMsg:
		if handled, cmd := m.handleGlobalKey(msg); handled {
			return m, cmd
		}
		if m.selectMode {
			switch msg.String() {
			case "q", "ctrl+c":
				seq := clearScreenSeq()
				if m.kitty && m.renderedImages {
					seq = deleteAllImagesSeq() + seq
				}
				return m, tea.Sequence(tea.Printf("%s", seq), tea.Quit)
			case "esc", "v":
				m.exitSelectMode()
				return m, nil
			case "j", "down", "l":
				m.selectDown()
				m.renderMessages()
				return m, nil
			case "k", "up", "h":
				m.selectUp()
				m.renderMessages()
				return m, nil
			case "V":
				m.toggleSelectRange()
				m.renderMessages()
				return m, nil
			case "y":
				m.yankSelected()
				return m, nil
			case "p":
				m.pasteYank()
				return m, nil
			case "e":
				return m, m.openEditorCmd()
			}
			return m, nil
		}
		if m.focusMessages {
			switch msg.String() {
			case "q", "ctrl+c":
				seq := clearScreenSeq()
				if m.kitty && m.renderedImages {
					seq = deleteAllImagesSeq() + seq
				}
				return m, tea.Sequence(tea.Printf("%s", seq), tea.Quit)
			case "tab":
				return m, m.cyclePaneFocus()
			case "j", "down":
				return m, m.startMessageSmoothScroll(paneScrollStep)
			case "k", "up":
				return m, m.startMessageSmoothScroll(-paneScrollStep)
			case "g", "home":
				m.viewport.GotoTop()
				return m, nil
			case "G", "end":
				m.viewport.GotoBottom()
				return m, nil
			case "ctrl+d", "pgdown":
				m.viewport.HalfPageDown()
				return m, nil
			case "ctrl+u", "pgup":
				m.viewport.HalfPageUp()
				return m, nil
			case "v":
				if m.enterSelectMode() {
					return m, nil
				}
			case "esc":
				return m, m.focusInput()
			}
			return m, nil
		}
		if m.isInputFocused() {
			if handled, cmd := m.updateInputModeKey(msg); handled {
				return m, cmd
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "q", "ctrl+c":
			seq := clearScreenSeq()
			if m.kitty && m.renderedImages {
				seq = deleteAllImagesSeq() + seq
			}
			return m, tea.Sequence(tea.Printf("%s", seq), tea.Quit)
		case "tab":
			return m, m.cyclePaneFocus()
		case "enter":
			if m.focusPicker && !m.focusAudience {
				if id, ok := m.selectedRoomID(); ok {
					return m, m.switchRoom(id)
				}
				return m, nil
			}
			if m.commandMode {
				cmd := m.runCommand(m.input.Value())
				m.exitCommandMode()
				return m, cmd
			}
			val := strings.TrimSpace(m.input.Value())
			if val != "" {
				m.input.SetValue("")
				return m, sendCmd(m.roomID, val, m.busChan)
			}
		case "pgup", "pgdown", "home", "end":
			if m.focusPicker && m.focusAudience {
				m.scrollAudience(msg.String())
				return m, nil
			}
		case "up":
			if m.focusPicker && !m.focusAudience {
				m.pickerUp()
				return m, nil
			}
		case "down":
			if m.focusPicker && !m.focusAudience {
				m.pickerDown()
				return m, nil
			}
		case "j":
			if m.focusPicker && m.focusAudience {
				return m, m.startAudienceSmoothScroll(paneScrollStep)
			}
			if m.focusPicker && !m.focusAudience {
				m.pickerDown()
				return m, nil
			}
		case "k":
			if m.focusPicker && m.focusAudience {
				return m, m.startAudienceSmoothScroll(-paneScrollStep)
			}
			if m.focusPicker && !m.focusAudience {
				m.pickerUp()
				return m, nil
			}
		case "g", "G", "ctrl+d", "ctrl+u":
			if m.focusPicker && m.focusAudience {
				m.scrollAudience(msg.String())
				return m, nil
			}
		case "d":
			if m.focusPicker && !m.focusAudience && !m.commandMode {
				m.startDeleteConfirm()
				return m, nil
			}
		case "y":
			if m.focusPicker && !m.focusAudience && m.deleteConfirmID != 0 {
				if m.confirmDeleteRecentRoom() && m.kitty && m.renderedImages {
					return m, tea.Printf("%s", deleteAllImagesSeq())
				}
				return m, nil
			}
		case "n":
			if m.deleteConfirmID != 0 {
				m.clearDeleteConfirm()
				return m, nil
			}
		case "esc":
			return m, m.focusInput()
		}
		return m, nil
	case danmuMsg:
		dm := getter.DanmuMsg(msg)
		m.appendMessage(dm)
		m.recordAudienceActivity(dm)
		cmds := []tea.Cmd{listenDanmuCmd(m.busChan)}
		if text := m.entryToastText(dm); text != "" {
			cmds = append(cmds, m.showToast(text))
		}
		if dm.UID > 0 && dm.Author != "" {
			if m.nameToUID[dm.Author] != dm.UID {
				m.nameToUID[dm.Author] = dm.UID
				m.renderMessages()
			}
			if m.kitty && !m.pendingUID[dm.UID] && m.avatarKeyBy[dm.UID] == "" {
				m.pendingUID[dm.UID] = true
				cmds = append(cmds, fetchUserInfoCmd(dm.UID))
			}
		}
		return m, tea.Batch(cmds...)
	case roomInfoMsg:
		m.roomInfo = getter.RoomInfo(msg)
		m.updateRecentRoomStatus(m.roomID, m.roomInfo.LiveStatus)
		cmds := []tea.Cmd{listenRoomInfoCmd(m.roomInfoChan)}
		if m.kitty && m.roomInfo.Uid != 0 {
			uid := int64(m.roomInfo.Uid)
			if uid != m.streamerUID && !m.pendingUID[uid] {
				m.streamerUID = uid
				m.pendingUID[uid] = true
				cmds = append(cmds, fetchUserInfoCmd(uid))
			}
		}
		if m.kitty && m.sidebarCover == "" {
			if key, url := pickRoomCover(m.roomInfo); key != "" {
				m.sidebarCover = key
				if _, ok := m.imageCache[key]; !ok {
					cmds = append(cmds, fetchImageCmd(key, url))
				}
			}
		}
		return m, tea.Batch(cmds...)
	case userInfoMsg:
		delete(m.pendingUID, msg.uid)
		cmds := []tea.Cmd{}
		if msg.err == nil {
			if msg.face != "" {
				key := imageKey(msg.face)
				m.avatarKeyBy[msg.uid] = key
				if _, ok := m.imageCache[key]; !ok {
					cmds = append(cmds, fetchImageCmd(key, msg.face))
				}
				if msg.uid == m.streamerUID {
					m.headerAvatar = key
				}
			}
			if msg.uid == m.streamerUID {
				coverURL := msg.liveCover
				if coverURL == "" {
					coverURL = msg.topPhoto
				}
				if coverURL != "" {
					key := imageKey(coverURL)
					m.sidebarCover = key
					if _, ok := m.imageCache[key]; !ok {
						cmds = append(cmds, fetchImageCmd(key, coverURL))
					}
				}
			}
			m.updateRecentRoomNameByUID(msg.uid, msg.name)
		}
		return m, tea.Batch(cmds...)
	case roomSummaryMsg:
		if msg.err == nil {
			updated := m.updateRecentRoomName(msg.roomID, msg.name)
			updated = m.updateRecentRoomStatus(msg.roomID, msg.live) || updated
			if msg.face != "" && m.kitty {
				key := roomAvatarKey(msg.roomID)
				m.roomAvatarKey[msg.roomID] = key
				if !m.loadRoomAvatar(msg.roomID) {
					updated = true
					cmds := []tea.Cmd{fetchRoomAvatarCmd(msg.roomID, msg.face)}
					if updated {
						m.renderMessages()
					}
					return m, tea.Batch(cmds...)
				}
			}
			if updated {
				m.renderMessages()
			}
		}
		return m, nil
	case roomStatusMsg:
		if msg.err == nil {
			if m.updateRecentRoomStatus(msg.roomID, msg.live) {
				m.renderMessages()
			}
		}
		return m, nil
	case roomsRefreshMsg:
		cmds := make([]tea.Cmd, 0, len(m.recentRooms)+1)
		for _, r := range m.recentRooms {
			cmds = append(cmds, fetchRoomStatusCmd(r.ID))
		}
		cmds = append(cmds, refreshRoomsTickCmd())
		return m, tea.Batch(cmds...)
	case imageMsg:
		if len(msg.data) != 0 {
			if msg.path != "" {
				if m.storeImageAtPath(msg.key, msg.path, msg.data) {
					m.renderMessages()
				}
			} else if m.storeImage(msg.key, msg.data) {
				m.renderMessages()
			}
		}
		return m, nil
	case editorFinishedMsg:
		m.editorPath = ""
		if msg.path != "" {
			if data, err := os.ReadFile(msg.path); err == nil {
				m.input.SetValue(strings.TrimRight(string(data), "\n"))
			}
			_ = os.Remove(msg.path)
		}
		return m, nil
	case toastExpiredMsg:
		if msg.seq == m.toastSeq {
			m.clearToast()
		}
		return m, nil
	case cookieValidatedMsg:
		if msg.err != nil {
			m.appendMessage(getter.DanmuMsg{
				Author:  "system",
				Content: "Cookie validation failed: " + msg.err.Error(),
				Type:    "NOTICE_MSG",
				Time:    time.Now(),
			})
		} else if !msg.ok {
			m.appendMessage(getter.DanmuMsg{
				Author:  "system",
				Content: "Configured cookie was rejected by Bilibili; update ~/.config/bili/config.toml if requests fail",
				Type:    "NOTICE_MSG",
				Time:    time.Now(),
			})
		}
		return m, nil
	case smoothScrollMsg:
		return m.handleSmoothScroll(msg)
	case startServicesMsg:
		m.stopGetter = msg.stop
		return m, nil
	case cursor.BlinkMsg:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *model) focusInput() tea.Cmd {
	m.focusPicker = false
	m.focusAudience = false
	m.focusMessages = false
	return m.input.Focus()
}

func smoothScrollTickCmd(target string, step int) tea.Cmd {
	return tea.Tick(smoothScrollDelay, func(time.Time) tea.Msg {
		return smoothScrollMsg{target: target, step: step}
	})
}

func (m *model) startMessageSmoothScroll(delta int) tea.Cmd {
	if delta == 0 {
		return nil
	}
	m.pendingMsgScroll += delta
	step := 1
	if delta < 0 {
		step = -1
	}
	return smoothScrollTickCmd("messages", step)
}

func (m *model) startAudienceSmoothScroll(delta int) tea.Cmd {
	if delta == 0 {
		return nil
	}
	m.pendingSideScroll += delta
	step := 1
	if delta < 0 {
		step = -1
	}
	return smoothScrollTickCmd("audience", step)
}

func (m *model) handleSmoothScroll(msg smoothScrollMsg) (tea.Model, tea.Cmd) {
	switch msg.target {
	case "messages":
		if m.pendingMsgScroll == 0 {
			return m, nil
		}
		step := msg.step
		if m.pendingMsgScroll < 0 {
			step = -1
		} else {
			step = 1
		}
		if step > 0 {
			m.viewport.LineDown(1)
			m.pendingMsgScroll--
		} else {
			m.viewport.LineUp(1)
			m.pendingMsgScroll++
		}
		if m.pendingMsgScroll != 0 {
			return m, smoothScrollTickCmd("messages", step)
		}
		return m, nil
	case "audience":
		if m.pendingSideScroll == 0 {
			return m, nil
		}
		step := msg.step
		if m.pendingSideScroll < 0 {
			step = -1
		} else {
			step = 1
		}
		if step > 0 {
			m.scrollAudienceStep(1)
			m.pendingSideScroll--
		} else {
			m.scrollAudienceStep(-1)
			m.pendingSideScroll++
		}
		if m.pendingSideScroll != 0 {
			return m, smoothScrollTickCmd("audience", step)
		}
		return m, nil
	default:
		return m, nil
	}
}

func validateCookieCmd() tea.Cmd {
	return func() tea.Msg {
		ok, err := config.ValidateCookie(config.Config.Cookie)
		return cookieValidatedMsg{ok: ok, err: err}
	}
}

func (m *model) handleGlobalKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "ctrl+t":
		m.cycleTheme()
		return true, nil
	default:
		return false, nil
	}
}

func (m *model) cycleTheme() {
	config.Config.Theme = nextThemeID(config.Config.Theme)
	m.styles = newStyles()
	applyThemeToInput(&m.input)
	m.renderMessages()
}

func nextThemeID(current int64) int64 {
	if current < 1 || current >= themeCount {
		return 1
	}
	return current + 1
}

func (m *model) focusRoomsPane() {
	m.focusPicker = true
	m.focusAudience = false
	m.focusMessages = false
	m.input.Blur()
}

func (m *model) focusAudiencePane() {
	m.focusPicker = true
	m.focusAudience = true
	m.focusMessages = false
	m.input.Blur()
}

func (m *model) focusMessagesPane() {
	m.focusPicker = false
	m.focusAudience = false
	m.focusMessages = true
	m.input.Blur()
}

func (m *model) isInputFocused() bool {
	return !m.focusPicker && !m.focusMessages
}

func (m *model) cyclePaneFocus() tea.Cmd {
	switch {
	case m.isInputFocused():
		m.focusRoomsPane()
		return nil
	case m.focusPicker && !m.focusAudience:
		m.focusAudiencePane()
		return nil
	case m.focusPicker && m.focusAudience:
		m.focusMessagesPane()
		return nil
	default:
		return m.focusInput()
	}
}

func (m *model) updateInputModeKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		seq := clearScreenSeq()
		if m.kitty && m.renderedImages {
			seq = deleteAllImagesSeq() + seq
		}
		return true, tea.Sequence(tea.Printf("%s", seq), tea.Quit)
	case "tab":
		return true, m.cyclePaneFocus()
	case "ctrl+f", "/":
		if !m.commandMode && strings.TrimSpace(m.input.Value()) == "" {
			m.enterSearchMode()
			return true, nil
		}
	case "enter":
		if m.searchMode {
			m.applySearch(m.input.Value())
			m.exitSearchMode()
			return true, nil
		}
		if m.commandMode {
			cmd := m.runCommand(m.input.Value())
			m.exitCommandMode()
			return true, cmd
		}
		val := strings.TrimSpace(m.input.Value())
		if val != "" {
			m.input.SetValue("")
			return true, sendCmd(m.roomID, val, m.busChan)
		}
		return true, nil
	case "pgup", "pgdown", "home", "end":
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return true, cmd
	case "esc":
		if m.searchMode {
			m.exitSearchMode()
			return true, nil
		}
		if m.searchQuery != "" {
			m.clearSearch()
			return true, nil
		}
		if m.deleteConfirmID != 0 {
			m.clearDeleteConfirm()
			return true, nil
		}
		if m.commandMode {
			m.exitCommandMode()
			return true, nil
		}
	case ":":
		if !m.commandMode && strings.TrimSpace(m.input.Value()) == "" {
			m.enterCommandMode()
			return true, nil
		}
	case "n":
		if m.searchQuery != "" && strings.TrimSpace(m.input.Value()) == "" {
			m.jumpSearch(1)
			return true, nil
		}
	case "N":
		if m.searchQuery != "" && strings.TrimSpace(m.input.Value()) == "" {
			m.jumpSearch(-1)
			return true, nil
		}
	}
	return false, nil
}
