package connect

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/proto"

	"gim/pkg/codec"
	pb "gim/pkg/protocol/pb/connectpb"
	"gim/pkg/protocol/pb/logicpb"
	"gim/pkg/util"
)

func TestTCPClient(t *testing.T) {
	runClient("tcp", "127.0.0.1:8001", 1, 1, 1)

}

func TestWSClient(t *testing.T) {
	runClient("ws", "ws://127.0.0.1:8002/ws", 1, 1, 1)

}

func TestGroupTCPClient(t *testing.T) {
	log.SetFlags(log.Lshortfile)

	go runClient("tcp", "127.0.0.1:8001", 1, 1, 1)
	go runClient("tcp", "127.0.0.1:8001", 2, 2, 1)
	select {}
}

type conn interface {
	write(buf []byte) error
	receive(handler func([]byte))
}

type tcpConn struct {
	conn   net.Conn
	reader *bufio.Reader
}

func newTCPConn(url string) (*tcpConn, error) {
	// demo "127.0.0.1:8001"
	conn, err := net.Dial("tcp", url)
	if err != nil {
		return nil, err
	}

	return &tcpConn{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}, nil
}

func (c *tcpConn) write(buf []byte) error {
	_, err := c.conn.Write(codec.Encode(buf))
	return err
}

func (c *tcpConn) receive(handler func([]byte)) {
	for {
		buf, err := codec.Decode(c.reader)
		if err != nil {
			log.Println(err)
			return
		}

		handler(buf)
	}
}

type wsConn struct {
	conn *websocket.Conn
}

func newWsConn(url string) (*wsConn, error) {
	// demo "ws://127.0.0.1:8002/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, http.Header{})
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println(string(bytes))
	return &wsConn{conn: conn}, nil
}

func (c *wsConn) write(buf []byte) error {
	return c.conn.WriteMessage(websocket.BinaryMessage, buf)
}

func (c *wsConn) receive(handler func([]byte)) {
	for {
		_, bytes, err := c.conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		handler(bytes)
	}
}

func jsonString(any any) string {
	bytes, _ := jsoniter.Marshal(any)
	return string(bytes)
}

type client struct {
	UserID   uint64
	DeviceID uint64
	Seq      uint64
	conn     conn
}

func runClient(network string, url string, userID, deviceID, seq uint64) {
	var conn conn
	var err error
	switch network {
	case "tcp":
		conn, err = newTCPConn(url)
	case "ws":
		conn, err = newWsConn(url)
	default:
		panic("unsupported network")
	}
	if err != nil {
		panic(err)
	}

	client := &client{
		UserID:   userID,
		DeviceID: deviceID,
		Seq:      seq,
		conn:     conn,
	}
	client.run()
}

func (c *client) run() {
	go c.conn.receive(c.handlePackage)
	c.signIn()
	c.syncTrigger()
	c.subscribeRoom()
	c.heartbeat()
}

func (c *client) info() string {
	return fmt.Sprintf("%-5d%-5d", c.UserID, c.DeviceID)
}

func (c *client) send(pt pb.Command, requestID int64, message proto.Message) {
	var packet = pb.Packet{
		Command:   pt,
		RequestId: requestID,
	}

	if message != nil {
		bytes, err := proto.Marshal(message)
		if err != nil {
			log.Println(c.info(), err)
			return
		}
		packet.Data = bytes
	}

	buf, err := proto.Marshal(&packet)
	if err != nil {
		log.Println(c.info(), err)
		return
	}

	err = c.conn.write(buf)
	if err != nil {
		log.Println(c.info(), err)
	}
}

func (c *client) signIn() {
	signIn := pb.SignInInput{
		UserId:   c.UserID,
		DeviceId: c.DeviceID,
		Token:    "0",
	}
	c.send(pb.Command_SIGN_IN, time.Now().UnixNano(), &signIn)
	log.Println(c.info(), "发送登录指令")
	time.Sleep(1 * time.Second)
}

func (c *client) syncTrigger() {
	c.send(pb.Command_SYNC, time.Now().UnixNano(), &pb.SyncInput{Seq: c.Seq})
	log.Println(c.info(), "开始同步")
}

func (c *client) heartbeat() {
	ticker := time.NewTicker(time.Minute * 5)
	for range ticker.C {
		c.send(pb.Command_HEARTBEAT, time.Now().UnixNano(), nil)
		fmt.Println(c.info(), "心跳发送")
	}
}

func (c *client) subscribeRoom() {
	var roomID uint64 = 1
	c.send(pb.Command_SUBSCRIBE_ROOM, 0, &pb.SubscribeRoomInput{
		RoomId: roomID,
		Seq:    0,
	})
	log.Println(c.info(), "订阅房间:", roomID)
}

func (c *client) handlePackage(bytes []byte) {
	var packet pb.Packet
	err := proto.Unmarshal(bytes, &packet)
	if err != nil {
		log.Println(err)
		return
	}

	switch packet.Command {
	case pb.Command_SIGN_IN:
		log.Println(c.info(), "登录响应:", jsonString(&packet))
	case pb.Command_HEARTBEAT:
		log.Println(c.info(), "心跳响应")
	case pb.Command_SYNC:
		log.Println(c.info(), "离线消息同步开始------")
		syncResp := pb.SyncOutput{}
		err := proto.Unmarshal(packet.Data, &syncResp)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(c.info(), "离线消息同步响应:code", packet.Code, "message:", packet.Message)
		for _, msg := range syncResp.Messages {
			log.Println(c.info(), util.MessageToString(msg))
			c.Seq = msg.Seq
		}

		ack := pb.MessageACK{
			DeviceAck:   c.Seq,
			ReceiveTime: util.UnixMilliTime(time.Now()),
		}
		c.send(pb.Command_MESSAGE, packet.RequestId, &ack)
		log.Println(c.info(), "离线消息同步结束------")
	case pb.Command_MESSAGE:
		msg := logicpb.Message{}
		err := proto.Unmarshal(packet.Data, &msg)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println(c.info(), util.MessageToString(&msg))
		c.Seq = msg.Seq
		ack := pb.MessageACK{
			DeviceAck:   msg.Seq,
			ReceiveTime: util.UnixMilliTime(time.Now()),
		}
		c.send(pb.Command_MESSAGE, packet.RequestId, &ack)
	case pb.Command_SUBSCRIBE_ROOM:
		log.Println(c.info(), "订阅房间响应", packet.Code, packet.Message)
	default:
		log.Println(c.info(), "switch other", &packet, len(bytes))
	}
}
