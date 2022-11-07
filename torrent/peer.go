package torrent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

type MsgId uint8

const (
	MsgChoke MsgId = iota
	MsgUnchoke
	MsgInterested
	MsgNotInterest
	MsgHave
	MsgBitfield
	MsgRequest
	MsgPiece
	MsgCancel
	MsgKeepalive
)

type PeerMsg struct {
	Id      MsgId
	Payload []byte
}

type PeerConn struct {
	net.Conn
	Choked  bool
	Field   Bitfield
	peer    PeerInfo
	peerID  [PeerIdLen]byte
	infoSHA [ShaLen]byte
}

func handshake(conn net.Conn, peerID [PeerIdLen]byte, infoSHA [ShaLen]byte) error {
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{})
	// send HandshakeMsg
	req := NewHandShakeMsg(infoSHA, peerID)
	_, err := req.WriteHandshake(conn)
	if err != nil {
		return fmt.Errorf("failed to send handshake: " + err.Error())
	}

	// read HandshakeMsg
	res, err := ReadHandshake(conn)
	if err != nil {
		return fmt.Errorf("failed to read handshake: " + err.Error())
	}

	// check HandshakeMsg
	if !bytes.Equal(res.InfoSHA[:], infoSHA[:]) {
		return fmt.Errorf("check handshake failed: " + string(res.InfoSHA[:]))
	}
	return nil
}

// ReadMsg parses a message from a stream. Returns `nil` on keep-alive message
func (c *PeerConn) ReadMsg() (*PeerMsg, error) {
	// read msg length
	lenBuf := make([]byte, 4)
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
	_, err = io.ReadFull(c, lenBuf)
	if err != nil {
		return nil, err
	}

	return &PeerMsg{
		Id:      MsgId(msgBuf[0]),
		Payload: msgBuf[1:],
	}, nil
}

// WriteMsg serializes a message into a buffer of the form
// <length prefix><message ID><payload>
// Interprets `nil` as a keep-alive message
func (c *PeerConn) WriteMsg(m *PeerMsg) (int, error) {
	if m == nil {
		return c.Write(make([]byte, 4))
	}
	length := uint32(len(m.Payload) + 1)
	buf := make([]byte, length+4)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.Id)
	copy(buf[5:], m.Payload)
	return c.Write(buf)
}
