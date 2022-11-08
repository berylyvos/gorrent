package torrent

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"testing"
)

func TestPeer(t *testing.T) {
	var peer PeerInfo
	peer.Ip = net.ParseIP("5.2.73.161")
	peer.Port = uint16(9091)

	file, _ := os.Open("../testfile/debian-iso.torrent")
	tf, _ := ParseFile(bufio.NewReader(file))

	var peerId [PeerIdLen]byte
	_, _ = rand.Read(peerId[:])

	conn, err := NewConn(peer, tf.InfoSHA, peerId)
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Printf("%+v\n", conn)
}
