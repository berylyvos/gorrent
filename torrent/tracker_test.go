package torrent

import (
	"crypto/rand"
	"fmt"
	"testing"
)

func TestRetrievePeers(t *testing.T) {
	tf, _ := Open("../testfile/debian-iso.torrent")

	var peerId [PeerIdLen]byte
	_, _ = rand.Read(peerId[:])

	peers := RetrievePeers(tf, peerId)
	for i, p := range peers {
		fmt.Printf("peer %d, Ip: %s, Port: %d\n", i, p.Ip, p.Port)
	}
}
