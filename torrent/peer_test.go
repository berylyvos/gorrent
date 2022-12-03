package torrent

import (
	"crypto/rand"
	"fmt"
	"net"
	"testing"
)

func TestPeer(t *testing.T) {
	var peer PeerInfo
	peer.Ip = net.ParseIP("5.2.73.161")
	peer.Port = uint16(9091)

	tf, _ := Open("../testfile/debian-iso.torrent")

	var peerId [PeerIdLen]byte
	_, _ = rand.Read(peerId[:])

	conn, err := NewConn(&peer, tf.InfoSHA, peerId)
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Printf("%+v\n", conn)
}
