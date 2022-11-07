package torrent

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"os"
	"testing"
)

func TestRetrievePeers(t *testing.T) {
	file, _ := os.Open("../testfile/debian-iso.torrent")
	tf, _ := ParseFile(bufio.NewReader(file))

	var peerId [PeerIdLen]byte
	_, _ = rand.Read(peerId[:])

	peers := RetrievePeers(tf, peerId)
	for i, p := range peers {
		fmt.Printf("peer %d, Ip: %s, Port: %d\n", i, p.Ip, p.Port)
	}
}
