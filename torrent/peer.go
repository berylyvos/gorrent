package torrent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

type MsgId uint8

const (
	// MsgChoke MsgUnchoke  Whether the remote peer has choked this client. When a peer chokes the
	// client, it is a notification that no requests will be answered until the client is unchoked.
	MsgChoke MsgId = iota
	MsgUnchoke

	// MsgInterested MsgNotInterest  Whether the remote peer is interested in something this client
	// has to offer. This is a notification that the remote peer will begin requesting blocks when
	// the client unchokes them.
	MsgInterested
	MsgNotInterest

	// MsgHave 'have' message's payload is a single number, the index which that downloader just
	// completed and checked the hash of. <len=0005><id=4><piece index>
	MsgHave

	// MsgBitfield 'bitfield' is only ever sent as the first message to show which blocks the
	// sender already downloaded. <len=0001+X><id=5><bitfield>
	MsgBitfield

	// MsgRequest To request a block. <len=0013><id=6><index><begin><length>
	MsgRequest

	// MsgPiece 'piece' is the response of 'request'. <len=0009+X><id=7><index><begin><block>
	MsgPiece

	// MsgCancel To end the download. <len=0013><id=8><index><begin><length>
	MsgCancel
)

type PeerMsg struct {
	Id      MsgId
	Payload []byte
}

type PeerConn struct {
	net.Conn
	Choked   bool
	BitField Bitfield
	peer     PeerInfo
	peerID   [PeerIdLen]byte
	infoSHA  [ShaLen]byte
}

func handshake(conn net.Conn, peerID [PeerIdLen]byte, infoSHA [ShaLen]byte) error {
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{})
	// send HandshakeMsg
	req := NewHandShakeMsg(infoSHA, peerID)
	_, err := req.WriteHandshake(conn)
	if err != nil {
		return fmt.Errorf("send handshake failed: " + err.Error())
	}

	// read HandshakeMsg
	res, err := ReadHandshake(conn)
	if err != nil {
		return fmt.Errorf("read handshake failed: " + err.Error())
	}

	// check HandshakeMsg
	if !bytes.Equal(res.InfoSHA[:], infoSHA[:]) {
		return fmt.Errorf("check handshake failed: " + string(res.InfoSHA[:]))
	}
	return nil
}

func fillBitfield(c *PeerConn) error {
	c.SetDeadline(time.Now().Add(5 * time.Second))
	defer c.SetDeadline(time.Time{})

	msg, err := c.ReadMsg()
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("expected bitfield")
	}
	if msg.Id != MsgBitfield {
		return fmt.Errorf("expected bitfield, get %d", msg.Id)
	}
	fmt.Println("fill bitfield: " + c.peer.Ip.String())
	c.BitField = msg.Payload
	return nil
}

const LenBytes uint8 = 4

// ReadMsg parses a message from a stream. Returns `nil` on keep-alive message
func (c *PeerConn) ReadMsg() (*PeerMsg, error) {
	// read msg length
	lenBuf := make([]byte, LenBytes)
	_, err := io.ReadFull(c, lenBuf)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)
	// keep-alive msg
	if length == 0 {
		return nil, nil
	}

	// read msg body
	msgBuf := make([]byte, length)
	_, err = io.ReadFull(c, msgBuf)
	if err != nil {
		return nil, err
	}

	return &PeerMsg{
		Id:      MsgId(msgBuf[0]),
		Payload: msgBuf[1:],
	}, nil
}

// WriteMsg serializes a message into a buffer of the form
// <4 bytes length><1 byte message ID><payload>
// Interprets `nil` as a keep-alive message
func (c *PeerConn) WriteMsg(m *PeerMsg) (int, error) {
	if m == nil {
		return c.Write(make([]byte, LenBytes))
	}
	length := uint32(len(m.Payload) + 1)
	buf := make([]byte, length+4)
	binary.BigEndian.PutUint32(buf[0:LenBytes], length)
	buf[4] = byte(m.Id)
	copy(buf[5:], m.Payload)
	return c.Write(buf)
}

func NewConn(peer PeerInfo, infoSHA [ShaLen]byte, peerId [PeerIdLen]byte) (*PeerConn, error) {
	// setup tcp connection
	addr := net.JoinHostPort(peer.Ip.String(), strconv.Itoa(int(peer.Port)))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("set tcp conn failed: " + addr)
	}
	// torrent peer to peer handshake
	err = handshake(conn, peerId, infoSHA)
	if err != nil {
		conn.Close()
		return nil, err
	}
	c := &PeerConn{
		Conn:    conn,
		Choked:  true,
		peer:    peer,
		peerID:  peerId,
		infoSHA: infoSHA,
	}
	// fill bitfield
	err = fillBitfield(c)
	if err != nil {
		return nil, fmt.Errorf("fill bitfield failed, " + err.Error())
	}
	return c, nil
}

func NewRequestMsg(index, offset, length int) *PeerMsg {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(offset))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &PeerMsg{MsgRequest, payload}
}

func GetHaveIndex(msg *PeerMsg) (int, error) {
	if msg.Id != MsgHave {
		return 0, fmt.Errorf("expected MsgHave (Id %d), got Id %d", MsgHave, msg.Id)
	}
	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("expected payload length 4, got length %d", len(msg.Payload))
	}
	return int(binary.BigEndian.Uint32(msg.Payload)), nil
}

func CopyPieceData(index int, buf []byte, msg *PeerMsg) (int, error) {
	if msg.Id != MsgPiece {
		return 0, fmt.Errorf("expected MsgPiece (Id %d), got Id %d", MsgPiece, msg.Id)
	}
	if len(msg.Payload) < 8 {
		return 0, fmt.Errorf("payload too short. %d < 8", len(msg.Payload))
	}
	pieceIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	if pieceIndex != index {
		return 0, fmt.Errorf("expected index %d, got %d", index, pieceIndex)
	}
	offset := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if offset >= len(buf) {
		return 0, fmt.Errorf("offset too high. %d >= %d", offset, len(buf))
	}
	data := msg.Payload[8:]
	if offset+len(data) > len(buf) {
		return 0, fmt.Errorf("data too large [%d] for offset %d with length %d", len(data), offset, len(buf))
	}
	copy(buf[offset:], data)
	return len(data), nil
}
