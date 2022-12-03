package torrent

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/berylyvos/gorrent/bencode"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	PeerIdLen            int = 20
	PeerPort             int = 7777
	IPLen                int = 4
	PortLen              int = 2
	PeerLen                  = IPLen + PortLen
	RetrievePeersTimeout int = 3
)

type PeerInfo struct {
	Ip   net.IP
	Port uint16
}

type TrackerResp struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func buildHTTPTrackerUrl(tf *TorrentFile, peerId [PeerIdLen]byte) ([]string, error) {
	var urls []string
	if len(tf.Announce) != 0 && isHTTPTrackerUrl(tf.Announce) {
		urls = append(urls, tf.Announce)
	}
	if len(tf.AnnounceList) != 0 {
		for _, an := range tf.AnnounceList {
			if isHTTPTrackerUrl(an) {
				urls = append(urls, an)
			}
		}
	}
	if len(urls) == 0 {
		return nil, errors.New("got no http announce url")
	}

	var res []string
	for _, u := range urls {
		baseUrl, err := url.Parse(u)
		if err != nil {
			fmt.Println("http announce url parse error: " + u)
			continue
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
		res = append(res, baseUrl.String())
	}
	return res, nil
}

func buildPeerInfo(peers []byte, peerChan chan *PeerInfo) {
	if len(peers)%PeerLen != 0 {
		fmt.Println("received malformed peers")
	}
	num := len(peers) / PeerLen
	for i := 0; i < num; i++ {
		offset := i * PeerLen
		peerChan <- &PeerInfo{
			Ip:   peers[offset : offset+IPLen],
			Port: binary.BigEndian.Uint16(peers[offset+IPLen : offset+PeerLen]),
		}
	}
}

func RetrievePeers(tf *TorrentFile, peerId [PeerIdLen]byte, peerMap *map[string]*PeerInfo) {
	httpTrackerUrls, err := buildHTTPTrackerUrl(tf, peerId)
	if err != nil {
		fmt.Println("build http tracker urls error: " + err.Error())
		return
	}

	peerChan := make(chan *PeerInfo)
	getPeersFromHTTPTrackers(httpTrackerUrls, peerChan)

	for {
		select {
		case p := <-peerChan:
			if _, ok := (*peerMap)[p.Ip.String()]; !ok {
				(*peerMap)[p.Ip.String()] = p
				fmt.Printf("peer [ip: %s, port: %d]\n", p.Ip, p.Port)
			}
		case <-time.After(3 * time.Second):
			return
		}
	}
}

func getPeersFromHTTPTrackers(trackerUrls []string, peerChan chan *PeerInfo) {
	for _, trackerUrl := range trackerUrls {
		go func(trackerUrl string) {
			cli := &http.Client{Timeout: time.Duration(RetrievePeersTimeout) * time.Second}
			resp, err := cli.Get(trackerUrl)
			if err != nil {
				fmt.Printf("failed to connect to tracker: %s error: %s\n", trackerUrl, err.Error())
				return
			}

			trackerResp := new(TrackerResp)
			err = bencode.Unmarshal(resp.Body, trackerResp)
			resp.Body.Close()
			if err != nil {
				fmt.Printf("tracker %s response error: %s\n", trackerUrl, err.Error())
				return
			}

			buildPeerInfo([]byte(trackerResp.Peers), peerChan)
		}(trackerUrl)

	}
}

func isHTTPTrackerUrl(url string) bool {
	return strings.HasPrefix(url, "http")
}

func isUDPTrackerUrl(url string) bool {
	return strings.HasPrefix(url, "udp")
}
