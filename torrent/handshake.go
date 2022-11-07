package torrent

import (
	"fmt"
	"io"
)

const (
	ReservedLen int = 8
	PreString       = "BitTorrent protocol"
	PreStrLen       = len(PreString)
	HsPreLen        = 1 + PreStrLen // 1 byte for the number of length of PreString
	HsMsgLen        = ReservedLen + ShaLen + PeerIdLen
)

type HandshakeMsg struct {
	PreStr  string
	InfoSHA [ShaLen]byte
	PeerID  [PeerIdLen]byte
}

func NewHandShakeMsg(infoSHA [ShaLen]byte, peerID [PeerIdLen]byte) *HandshakeMsg {
	return &HandshakeMsg{
		PreStr:  PreString,
		InfoSHA: infoSHA,
		PeerID:  peerID,
	}
}

func (msg *HandshakeMsg) WriteHandshake(w io.Writer) (int, error) {
	buf := make([]byte, HsPreLen+HsMsgLen)
	buf[0] = byte(len(msg.PreStr)) // 0x13
	curr := 1
	curr += copy(buf[curr:], msg.PreStr)
	curr += copy(buf[curr:], make([]byte, ReservedLen))
	curr += copy(buf[curr:], msg.InfoSHA[:])
	curr += copy(buf[curr:], msg.PeerID[:])
	return w.Write(buf)
}

func ReadHandshake(r io.Reader) (*HandshakeMsg, error) {
	lenBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lenBuf)
	if err != nil {
		return nil, err
	}
	preLen := int(lenBuf[0])
	if preLen == 0 {
		return nil, fmt.Errorf("handshake prelen cannot not be 0")
	}

	msgBuf := make([]byte, HsMsgLen+preLen)
	_, err = io.ReadFull(r, msgBuf)
	if err != nil {
		return nil, err
	}

	var infoSHA [ShaLen]byte
	var peerId [PeerIdLen]byte
	copy(infoSHA[:], msgBuf[preLen+ReservedLen:preLen+ReservedLen+ShaLen])
	copy(peerId[:], msgBuf[preLen+ReservedLen+ShaLen:])

	return &HandshakeMsg{
		PreStr:  string(msgBuf[0:preLen]),
		InfoSHA: infoSHA,
		PeerID:  peerId,
	}, nil
}
