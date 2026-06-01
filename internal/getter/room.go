package getter

import (
	"fmt"
	"strings"
	"time"

	"github.com/noahlias/bili-live-tui/internal/config"

	"github.com/asmcos/requests"

	"github.com/tidwall/gjson"
)

func (d *DanmuClient) syncRoomInfo(roomInfoChan chan RoomInfo, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
		}
		if d.isClosed {
			return
		}
		startedAt := time.Now()

		roomInfoApi := fmt.Sprintf("https://api.live.bilibili.com/room/v1/room/get_info?room_id=%d", d.roomID)
		roomInfo := new(RoomInfo)
		roomInfo.OnlineRankUsers = make([]OnlineRankUser, 0)
		r1, err1 := requests.Get(roomInfoApi)
		if err1 == nil {
			roomInfo.RoomId = int(d.roomID)
			roomInfo.Uid = int(gjson.Get(r1.Text(), "data.uid").Int())
			roomInfo.Title = gjson.Get(r1.Text(), "data.title").String()
			roomInfo.AreaName = gjson.Get(r1.Text(), "data.area_name").String()
			roomInfo.ParentAreaName = gjson.Get(r1.Text(), "data.parent_area_name").String()
			roomInfo.Online = gjson.Get(r1.Text(), "data.online").Int()
			roomInfo.Attention = gjson.Get(r1.Text(), "data.attention").Int()
			roomInfo.Background = gjson.Get(r1.Text(), "data.background").String()
			roomInfo.UserCover = gjson.Get(r1.Text(), "data.user_cover").String()
			roomInfo.Keyframe = gjson.Get(r1.Text(), "data.keyframe").String()
			roomInfo.LiveStatus = gjson.Get(r1.Text(), "data.live_status").Int()
			roomInfo.Time = formatLiveDuration(roomInfo.LiveStatus, gjson.Get(r1.Text(), "data.live_time").String(), time.Now())
		}

		if roomInfo.Uid != 0 {
			if d.shouldRefreshOnlineRank() {
				users, total, err := fetchAllOnlineRankUsers(int64(d.roomID), int64(roomInfo.Uid))
				if err == nil || len(users) > 0 {
					d.rankUsers = users
					d.rankTotal = total
					d.rankFetchedAt = time.Now()
				}
			}
			roomInfo.OnlineRankTotal = d.rankTotal
			roomInfo.OnlineRankUsers = append(roomInfo.OnlineRankUsers, d.rankUsers...)
			if d.shouldRefreshAudience() {
				users, err := fetchKnownAudienceUsers(int64(d.roomID), int64(roomInfo.Uid), d.rankUsers)
				if err == nil || len(users) > 0 {
					d.audienceUsers = users
					d.audienceAt = time.Now()
				}
			}
			roomInfo.AudienceUsers = append(roomInfo.AudienceUsers, d.audienceUsers...)
		}
		roomInfo.APILatencyMs = time.Since(startedAt).Milliseconds()
		roomInfo.WSLatencyMs = d.wsLatencyMs.Load()

		roomInfoChan <- *roomInfo
		select {
		case <-done:
			return
		case <-time.After(30 * time.Second):
		}
	}
}

func formatLiveDuration(liveStatus int64, liveTime string, now time.Time) string {
	if liveStatus != 1 {
		return ""
	}
	liveTime = strings.TrimSpace(liveTime)
	if liveTime == "" {
		return ""
	}
	loc := time.FixedZone("UTC+8", 8*60*60)
	startedAt, err := time.ParseInLocation("2006-01-02 15:04:05", liveTime, loc)
	if err != nil || startedAt.IsZero() {
		return ""
	}
	seconds := int64(now.In(loc).Sub(startedAt) / time.Second)
	if seconds < 0 {
		return ""
	}
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%d天%d时%d分", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d时%d分", hours, minutes)
	}
	return fmt.Sprintf("%d分", minutes)
}

func (d *DanmuClient) shouldRefreshOnlineRank() bool {
	if len(d.rankUsers) == 0 {
		return true
	}
	return time.Since(d.rankFetchedAt) >= 2*time.Minute
}

func (d *DanmuClient) shouldRefreshAudience() bool {
	if len(d.audienceUsers) == 0 {
		return true
	}
	return time.Since(d.audienceAt) >= 2*time.Minute
}

func supervisor(roomID int64, busChan chan DanmuMsg, roomInfoChan chan RoomInfo, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
		}
		dc := DanmuClient{
			roomID:        uint32(roomID),
			unzlibChannel: make(chan []byte, 100),
		}
		if err := dc.connect(); err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		go dc.getHistory(busChan)
		go dc.receiveRawMsg(busChan, done)
		go dc.syncRoomInfo(roomInfoChan, done)
		go dc.heartBeat(busChan, done)

		for {
			select {
			case <-done:
				dc.close()
				return
			default:
			}
			time.Sleep(1 * time.Second)
			if dc.isClosed {
				busChan <- DanmuMsg{
					Author:  "system",
					Content: "弹幕服务器已断开，正在重连 :)",
					Type:    "NOTICE_MSG",
					Time:    time.Now(),
				}
				time.Sleep(1 * time.Second)
				break
			}
		}
	}
}

func Run(busChan chan DanmuMsg, roomInfoChan chan RoomInfo) {
	RunWithRoom(config.Config.RoomId, busChan, roomInfoChan)
}

func RunWithRoom(roomID int64, busChan chan DanmuMsg, roomInfoChan chan RoomInfo) func() {
	done := make(chan struct{})
	go supervisor(roomID, busChan, roomInfoChan, done)
	return func() {
		close(done)
	}
}

func ResetHistory(roomID int64) {
	historyMu.Lock()
	delete(historyByRoom, uint32(roomID))
	historyMu.Unlock()
}

func GetRoomSummary(roomID int64) (RoomSummary, error) {
	roomInfoApi := fmt.Sprintf("https://api.live.bilibili.com/room/v1/room/get_info?room_id=%d", roomID)
	r, err := requests.Get(roomInfoApi)
	if err != nil {
		return RoomSummary{}, err
	}
	if gjson.Get(r.Text(), "code").Int() != 0 {
		return RoomSummary{}, fmt.Errorf("room info response error")
	}
	uid := gjson.Get(r.Text(), "data.uid").Int()
	if uid <= 0 {
		return RoomSummary{}, fmt.Errorf("room uid not found")
	}
	return RoomSummary{
		UID:        uid,
		LiveStatus: gjson.Get(r.Text(), "data.live_status").Int(),
		Title:      gjson.Get(r.Text(), "data.title").String(),
	}, nil
}

func GetRoomUID(roomID int64) (int64, error) {
	summary, err := GetRoomSummary(roomID)
	if err != nil {
		return 0, err
	}
	return summary.UID, nil
}
