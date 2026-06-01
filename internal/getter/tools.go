package getter

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/gorilla/websocket"
)

func zlibUnCompress(compressSrc []byte) ([]byte, error) {
	b := bytes.NewReader(compressSrc)
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func brotliUnCompress(compressSrc []byte) ([]byte, error) {
	r := brotli.NewReader(bytes.NewReader(compressSrc))
	return io.ReadAll(r)
}

func (d *DanmuClient) sendPackage(packetlen uint32, magic uint16, ver uint16, typeID uint32, param uint32, data []byte) (err error) {
	if d.conn == nil || d.conn.UnderlyingConn() == nil {
		return fmt.Errorf("danmu websocket not connected")
	}
	packetHead := new(bytes.Buffer)

	if packetlen == 0 {
		packetlen = uint32(len(data) + 16)
	}
	var pdata = []interface{}{
		packetlen,
		magic,
		ver,
		typeID,
		param,
	}

	// 将包的头部信息以大端序方式写入字节数组
	for _, v := range pdata {
		if err = binary.Write(packetHead, binary.BigEndian, v); err != nil {
			fmt.Println("binary.Write err: ", err)
			return
		}
	}

	// 将包内数据部分追加到数据包内
	sendData := append(packetHead.Bytes(), data...)

	// fmt.Println("本次发包消息为：", sendData)

	if err = d.conn.WriteMessage(websocket.BinaryMessage, sendData); err != nil {
		fmt.Println("conn.Write err: ", err)
		return
	}

	return
}

type wsPacket struct {
	packetLen uint32
	headerLen uint16
	ver       uint16
	op        uint32
	seq       uint32
	body      []byte
}

func parsePackets(data []byte) ([]wsPacket, error) {
	packets := make([]wsPacket, 0, 4)
	offset := 0
	for offset+16 <= len(data) {
		packetLen := binary.BigEndian.Uint32(data[offset : offset+4])
		if packetLen < 16 || offset+int(packetLen) > len(data) {
			break
		}
		headerLen := binary.BigEndian.Uint16(data[offset+4 : offset+6])
		ver := binary.BigEndian.Uint16(data[offset+6 : offset+8])
		op := binary.BigEndian.Uint32(data[offset+8 : offset+12])
		seq := binary.BigEndian.Uint32(data[offset+12 : offset+16])
		bodyStart := offset + int(headerLen)
		bodyEnd := offset + int(packetLen)
		if bodyStart > bodyEnd || bodyEnd > len(data) {
			break
		}
		packets = append(packets, wsPacket{
			packetLen: packetLen,
			headerLen: headerLen,
			ver:       ver,
			op:        op,
			seq:       seq,
			body:      data[bodyStart:bodyEnd],
		})
		offset += int(packetLen)
	}
	if len(packets) == 0 {
		return nil, fmt.Errorf("no packets parsed")
	}
	return packets, nil
}

func decodeDanmuMessages(raw []byte) ([][]byte, error) {
	packets, err := parsePackets(raw)
	if err != nil {
		return nil, err
	}
	var bodies [][]byte
	for _, p := range packets {
		switch p.ver {
		case 0, 1:
			if p.op == 5 && len(p.body) > 0 {
				bodies = append(bodies, p.body)
			}
		case 2:
			if p.op != 5 || len(p.body) == 0 {
				continue
			}
			uz, err := zlibUnCompress(p.body)
			if err != nil {
				continue
			}
			sub, err := parsePackets(uz)
			if err != nil {
				continue
			}
			for _, sp := range sub {
				if sp.op == 5 && len(sp.body) > 0 {
					bodies = append(bodies, sp.body)
				}
			}
		case 3:
			if p.op != 5 || len(p.body) == 0 {
				continue
			}
			uz, err := brotliUnCompress(p.body)
			if err != nil {
				continue
			}
			sub, err := parsePackets(uz)
			if err != nil {
				continue
			}
			for _, sp := range sub {
				if sp.op == 5 && len(sp.body) > 0 {
					bodies = append(bodies, sp.body)
				}
			}
		}
	}
	return bodies, nil
}
