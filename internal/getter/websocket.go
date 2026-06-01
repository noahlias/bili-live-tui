package getter

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	myhttp "github.com/BYT0723/go-tools/http"

	"github.com/asmcos/requests"
	"github.com/gorilla/websocket"

	"github.com/tidwall/gjson"
)

func (d *DanmuClient) connect() (err error) {
	var (
		uid    uint32
		body   []byte
		header = buildBaseHeader()
	)

	_, body, err = myhttp.Get("https://api.bilibili.com/x/web-interface/nav", header, nil)
	if err != nil {
		return err
	}
	uid = uint32(gjson.GetBytes(body, "data.mid").Int())
	updateWbiFromNav(body)

	query, err := signWbiParams(map[string]string{
		"id": fmt.Sprintf("%d", d.roomID),
	}, header)
	if err != nil {
		return err
	}
	_, body, err = myhttp.Get("https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuInfo?"+query, header, nil)
	if err != nil {
		return err
	}

	token := gjson.GetBytes(body, "data.token").String()
	type wsHost struct {
		host    string
		wssPort int
	}
	hostList := []wsHost{}
	gjson.GetBytes(body, "data.host_list").ForEach(func(key, value gjson.Result) bool {
		host := value.Get("host").String()
		if host == "" {
			return true
		}
		port := int(value.Get("wss_port").Int())
		if port == 0 {
			port = 443
		}
		hostList = append(hostList, wsHost{host: host, wssPort: port})
		return true
	})
	if len(hostList) == 0 {
		return fmt.Errorf("danmu host list empty")
	}
	hsInfo := handShakeInfo{
		UID:      uid,
		Roomid:   d.roomID,
		Protover: 3,
		Platform: "web",
		Type:     2,
		Key:      token,
	}

	header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36")
	header.Set("Accept", "*/*")
	header.Set("Accept-Language", "zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2")
	header.Set("Accept-Encoding", "gzip, deflate, br")
	header.Set("Origin", "https://live.bilibili.com")
	header.Set("Pragma", "no-cache")
	header.Set("Cache-Control", "no-cache")
	header.Set("Custom-Header", "CustomValue")
	var dialErr error
	d.conn = nil
	for _, h := range hostList {
		d.conn, _, dialErr = websocket.DefaultDialer.Dial(fmt.Sprintf("wss://%s:%d/sub", h.host, h.wssPort), header)
		if dialErr != nil {
			continue
		}
		break
	}
	if d.conn == nil {
		if dialErr != nil {
			return dialErr
		}
		return fmt.Errorf("failed to connect danmu websocket")
	}
	body, err = json.Marshal(hsInfo)
	if err != nil {
		return
	}

	err = d.sendPackage(0, 16, 1, 7, 1, body)
	return
}

var historyMu sync.Mutex

var historyByRoom = make(map[uint32]bool)

func (d *DanmuClient) getHistory(busChan chan DanmuMsg) {
	historyMu.Lock()
	if historyByRoom[d.roomID] {
		historyMu.Unlock()
		return
	}
	historyByRoom[d.roomID] = true
	historyMu.Unlock()

	historyApi := fmt.Sprintf("https://api.live.bilibili.com/xlive/web-room/v1/dM/gethistory?roomid=%d", d.roomID)
	r, err := requests.Get(historyApi)
	if err != nil {
		return
	}

	histories := gjson.Get(r.Text(), "data.room").Array()
	for _, history := range histories {
		t, _ := time.Parse("2006-01-02 15:04:05", history.Get("timeline").String())
		danmu := DanmuMsg{
			Author:  history.Get("nickname").String(),
			Content: history.Get("text").String(),
			Type:    "DANMU_MSG",
			Time:    t,
			UID:     history.Get("uid").Int(),
		}
		busChan <- danmu
	}
}

func (d *DanmuClient) heartBeat(msgChan chan DanmuMsg, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
		}
		if d.isClosed {
			return
		}
		obj := []byte("5b6f626a656374204f626a6563745d")
		if err := d.sendPackage(0, 16, 1, 2, 1, obj); err != nil {
			msgChan <- DanmuMsg{
				// Author
				// Content string
				// Type    string
			}
			continue
		}
		d.lastHeartbeatSentNs.Store(time.Now().UnixNano())
		time.Sleep(30 * time.Second)
	}
}

func (d *DanmuClient) receiveRawMsg(busChan chan DanmuMsg, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
		}
		if d.isClosed {
			return
		}
		_, rawMsg, err := d.conn.ReadMessage()
		if err != nil {
			d.isClosed = true
		}
		d.updateWebsocketLatency(rawMsg)
		msgBodies, err := decodeDanmuMessages(rawMsg)
		if err != nil {
			continue
		}
		for _, body := range msgBodies {
			js := new(receivedInfo)
			if err := json.Unmarshal(body, js); err != nil {
				continue
			}
			m := DanmuMsg{}
			switch js.Cmd {
			case "COMBO_SEND":
				m.Author = mapString(js.Data, "uname")
				m.Content = fmt.Sprintf("送给 %s %d 个 %s", mapString(js.Data, "r_uname"), mapInt(js.Data, "combo_num"), mapString(js.Data, "gift_name"))
				m.UID = mapInt64(js.Data, "uid")
			case "DANMU_MSG":
				m.Author, m.UID = parseDanmuUser(js.Info)
				m.Content = parseDanmuContent(js.Info)
			case "GUARD_BUY":
				m.Author = mapString(js.Data, "username")
				m.Content = fmt.Sprintf("购买了 %s", mapString(js.Data, "giftName"))
				m.UID = mapInt64(js.Data, "uid")
			case "INTERACT_WORD":
				m.Author = mapString(js.Data, "uname")
				m.Content = interactionContent(js.Data)
				m.UID = mapInt64(js.Data, "uid")
			case "SEND_GIFT":
				m.Author = mapString(js.Data, "uname")
				m.Content = fmt.Sprintf("投喂了 %d 个 %s", mapInt(js.Data, "num"), mapString(js.Data, "giftName"))
				m.UID = mapInt64(js.Data, "uid")
			case "USER_TOAST_MSG":
				m.Author = "system"
				m.Content = mapString(js.Data, "toast_msg")
			case "NOTICE_MSG":
				m.Author = "system"
				m.Content = js.MsgSelf
			default: // "LIVE" "ACTIVITY_BANNER_UPDATE_V2" "ONLINE_RANK_COUNT" "ONLINE_RANK_TOP3" "ONLINE_RANK_V2" "PANEL" "PREPARING" "WIDGET_BANNER" "LIVE_INTERACTIVE_GAME"
				continue
			}
			m.Type = js.Cmd
			m.Time = time.Now()
			busChan <- m
		}
	}
}

func (d *DanmuClient) updateWebsocketLatency(raw []byte) {
	if len(raw) == 0 {
		return
	}
	packets, err := parsePackets(raw)
	if err != nil {
		return
	}
	sentNs := d.lastHeartbeatSentNs.Load()
	if sentNs <= 0 {
		return
	}
	for _, packet := range packets {
		if packet.op != 3 {
			continue
		}
		latency := time.Since(time.Unix(0, sentNs)).Milliseconds()
		if latency < 0 {
			return
		}
		d.wsLatencyMs.Store(latency)
		return
	}
}

func (d *DanmuClient) close() {
	d.isClosed = true
	if d.conn != nil && d.conn.UnderlyingConn() != nil {
		d.conn.Close()
	}
}
