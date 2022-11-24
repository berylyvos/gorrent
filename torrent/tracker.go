package torrent

import (
	"encoding/binary"
	"fmt"
	"github.com/berylyvos/gorrent/bencode"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	PeerIdLen            int = 20
	PeerPort             int = 7777
	IPLen                int = 4
	PortLen              int = 2
	PeerLen                  = IPLen + PortLen
	RetrievePeersTimeout int = 15
)

type PeerInfo struct {
	Ip   net.IP
	Port uint16
}

type TrackerResp struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func buildTrackerUrl(tf *TorrentFile, peerId [PeerIdLen]byte) (string, error) {
	baseUrl, err := url.Parse(tf.Announce)
	if err != nil {
		fmt.Println("announce url parse error: " + tf.Announce)
		return "", err
	}

	params := url.Values{
		"info_hash":  []string{string(tf.InfoSHA[:])},
		"peer_id":    []string{string(peerId[:])},
		"port":       []string{strconv.Itoa(PeerPort)},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(tf.FileLen)},
	}

	baseUrl.RawQuery = params.Encode()
	return baseUrl.String(), nil
}

func buildPeerInfo(peers []byte) []PeerInfo {
	if len(peers)%PeerLen != 0 {
		fmt.Println("received malformed peers")
	}
	num := len(peers) / PeerLen
	peerInfo := make([]PeerInfo, num)
	for i := 0; i < num; i++ {
		offset := i * PeerLen
		peerInfo[i].Ip = peers[offset : offset+IPLen]
		peerInfo[i].Port = binary.BigEndian.Uint16(peers[offset+IPLen : offset+PeerLen])
	}
	return peerInfo
}

func RetrievePeers(tf *TorrentFile, peerId [PeerIdLen]byte) []PeerInfo {
	trackerUrl, err := buildTrackerUrl(tf, peerId)
	if err != nil {
		fmt.Println("build tracker url error: " + err.Error())
		return nil
	}

	cli := &http.Client{Timeout: time.Duration(RetrievePeersTimeout) * time.Second}
	resp, err := cli.Get(trackerUrl)
	if err != nil {
		fmt.Println("failed to connect to tracker: " + err.Error())
		return nil
	}
	defer resp.Body.Close()

	trackerResp := new(TrackerResp)
	err = bencode.Unmarshal(resp.Body, trackerResp)
	if err != nil {
		fmt.Println("tracker response error: " + err.Error())
		return nil
	}

	return buildPeerInfo([]byte(trackerResp.Peers))
}
