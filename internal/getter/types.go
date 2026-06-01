package getter

import (
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type DanmuClient struct {
	roomID              uint32
	conn                *websocket.Conn
	unzlibChannel       chan []byte
	isClosed            bool
	rankUsers           []OnlineRankUser
	rankTotal           int64
	rankFetchedAt       time.Time
	audienceUsers       []AudienceUser
	audienceAt          time.Time
	adminUsers          []AudienceUser
	adminAt             time.Time
	lastHeartbeatSentNs atomic.Int64
	wsLatencyMs         atomic.Int64
}

type OnlineRankUser struct {
	UID         int64
	Name        string
	Face        string
	Score       int64
	Rank        int64
	MedalName   string
	MedalLevel  int64
	GuardLevel  int64
	WealthLevel int64
}

type AudienceUser struct {
	UID         int64
	Name        string
	Face        string
	Sources     []string
	Rank        int64
	Score       int64
	MedalName   string
	MedalLevel  int64
	GuardLevel  int64
	WealthLevel int64
}

type RoomInfo struct {
	RoomId          int
	Uid             int
	Title           string
	ParentAreaName  string
	AreaName        string
	Online          int64
	Attention       int64
	Time            string
	Background      string
	UserCover       string
	Keyframe        string
	LiveStatus      int64
	OnlineRankTotal int64
	OnlineRankUsers []OnlineRankUser
	AudienceUsers   []AudienceUser
	AdminUsers      []AudienceUser
	APILatencyMs    int64
	WSLatencyMs     int64
}

type DanmuMsg struct {
	Author  string
	Content string
	Type    string
	Time    time.Time
	UID     int64
}

type receivedInfo struct {
	Cmd        string                 `json:"cmd"`
	Data       map[string]interface{} `json:"data"`
	Info       []interface{}          `json:"info"`
	Full       map[string]interface{} `json:"full"`
	Half       map[string]interface{} `json:"half"`
	Side       map[string]interface{} `json:"side"`
	RoomID     uint32                 `json:"roomid"`
	RealRoomID uint32                 `json:"real_roomid"`
	MsgCommon  string                 `json:"msg_common"`
	MsgSelf    string                 `json:"msg_self"`
	LinkUrl    string                 `json:"link_url"`
	MsgType    string                 `json:"msg_type"`
	ShieldUID  string                 `json:"shield_uid"`
	BusinessID string                 `json:"business_id"`
	Scatter    map[string]interface{} `json:"scatter"`
}

type handShakeInfo struct {
	UID      uint32 `json:"uid"`
	Roomid   uint32 `json:"roomid"`
	Protover uint8  `json:"protover"`
	Buvid    string `json:"buvid"`
	Platform string `json:"platform"`
	Type     uint8  `json:"type"`
	Key      string `json:"key"`
}

type RoomSummary struct {
	UID        int64
	LiveStatus int64
	Title      string
}

type onlineRankPage struct {
	total int64
	users []OnlineRankUser
}

type onlineRankUsersFetcher func(roomID int64, uid int64) ([]OnlineRankUser, int64, error)
