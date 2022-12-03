package torrent

import (
	"crypto/rand"
	"testing"
)

func TestRetrievePeers(t *testing.T) {
	tf, _ := Open("../testfile/nope.torrent")

	var peerId [PeerIdLen]byte
	_, _ = rand.Read(peerId[:])

	peerMap := make(map[string]*PeerInfo)
	RetrievePeers(tf, peerId, &peerMap)
}
