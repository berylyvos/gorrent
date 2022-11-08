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
	// MsgChoke MsgUnchoke MsgInterested MsgNotInterest choking algorithm relative shit
	MsgChoke MsgId = iota
	MsgUnchoke
	MsgInterested
	MsgNotInterest

	// MsgHave 'have' message's payload is a single number, the index which that downloader
	// just completed and checked the hash of. (To notify the peer that downloaded from)
	MsgHave

	// MsgBitfield 'bitfield' is only ever sent as the first message to show which block the
	// sender already downloaded.
	MsgBitfield

	// MsgRequest To request a block. 'request' message contain an index, begin, and length.
	MsgRequest

	// MsgPiece 'piece' is the response of 'request'
	MsgPiece

	// MsgCancel 'cancel' end the download. the same payload as 'request'
	MsgCancel
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
	c.Field = msg.Payload
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
