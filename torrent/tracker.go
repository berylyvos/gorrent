package torrent

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/berylyvos/gorrent/bencode"
	"math/rand"
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
	RetrievePeersTimeout int = 5
)

const UDPTrackerProtocolID = 0x41727101980

// UDP Tracker protocol action type
const (
	ActionConnect = iota
	ActionAnnounce
	ActionScrape
	ActionError
)

type PeerInfo struct {
	Ip   net.IP
	Port uint16
}

type HTTPTrackerResp struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

type UDPTracker struct {
	Host string
	IP   net.IP
	Port int
}

func buildHTTPTrackerUrls(tf *TorrentFile, peerId [PeerIdLen]byte) ([]string, error) {
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

func buildUDPTrackers(tf *TorrentFile) ([]UDPTracker, error) {
	var trackers []UDPTracker
	for _, ann := range tf.AnnounceList {
		if !isUDPTrackerUrl(ann) {
			continue
		}
		tr := UDPTracker{}
		urlStr := strings.Split(strings.Split(ann, "/")[2], ":")
		tr.Host = urlStr[0]
		tr.Port, _ = strconv.Atoi(urlStr[1])
		ips, err := net.LookupIP(urlStr[0])
		if err != nil {
			fmt.Printf("%v host look up ip error: %v\n", tr.Host, err)
			continue
		}
		tr.IP = ips[0]
		trackers = append(trackers, tr)
	}
	return trackers, nil
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
	httpTrackerUrls, err := buildHTTPTrackerUrls(tf, peerId)
	if err != nil {
		fmt.Println("build http tracker urls error: " + err.Error())
	}
	udpTrackers, err := buildUDPTrackers(tf)
	if err != nil {
		fmt.Println("build udp tracker urls error: " + err.Error())
	}

	peerChan := make(chan *PeerInfo)
	getPeersFromHTTPTrackers(httpTrackerUrls, peerChan)
	getPeersFromUDPTrackers(tf, udpTrackers, peerId, peerChan)

	for {
		select {
		case p := <-peerChan:
			if _, ok := (*peerMap)[p.Ip.String()]; !ok {
				(*peerMap)[p.Ip.String()] = p
				fmt.Printf("peer [ip: %s, port: %d]\n", p.Ip, p.Port)
			}
		case <-time.After(time.Duration(RetrievePeersTimeout) * time.Second):
			close(peerChan)
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

			trackerResp := new(HTTPTrackerResp)
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

func getPeersFromUDPTrackers(tf *TorrentFile, udpTrackers []UDPTracker, peerId [PeerIdLen]byte, peerChan chan *PeerInfo) {
	for _, tr := range udpTrackers {
		go connect(tf, tr, peerId, peerChan)
	}
}

func connect(tf *TorrentFile, tracker UDPTracker, peerId [PeerIdLen]byte, peerChan chan *PeerInfo) {
	socket, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   tracker.IP,
		Port: tracker.Port,
	})
	if err != nil {
		fmt.Printf("%v connect dial error: %v\n", tracker.Host, err)
		return
	}
	defer func() { _ = socket.Close() }()

	// connect request:
	// Offset  Size            Name            Value
	// 0       64-bit integer  protocol_id     0x41727101980 // magic constant
	// 8       32-bit integer  action          0 // connect
	// 12      32-bit integer  transaction_id
	// 16
	payload := make([]byte, 16)
	binary.BigEndian.PutUint64(payload[0:8], uint64(UDPTrackerProtocolID))
	binary.BigEndian.PutUint32(payload[8:12], uint32(ActionConnect))
	binary.BigEndian.PutUint32(payload[12:16], uint32(genTransactionID()))
	_, err = socket.Write(payload)
	if err != nil {
		fmt.Printf("%v connect write payload error: %v\n", tracker.Host, err)
		return
	}
	data := make([]byte, 16)
	_ = socket.SetReadDeadline(time.Now().Add(time.Duration(RetrievePeersTimeout) * time.Second))
	n, remoteAddr, err := socket.ReadFromUDP(data)
	if err != nil {
		fmt.Printf("%v connect read from udp error: %v\n", tracker.Host, err)
		return
	}
	// connect response:
	// 0       32-bit integer  action          0 // connect
	// 4       32-bit integer  transaction_id
	// 8       64-bit integer  connection_id
	// 16
	if binary.BigEndian.Uint32(data[:4]) == ActionError {
		fmt.Printf("%v connect response error\n", tracker.Host)
		return
	}

	fmt.Printf("connect recv:%v addr:%v count:%v\n", data[:n], remoteAddr, n)
	announce(tf, binary.BigEndian.Uint64(data[8:16]), peerId, tracker, peerChan)
}

func announce(tf *TorrentFile, connId uint64, peerId [PeerIdLen]byte, tracker UDPTracker, peerChan chan *PeerInfo) {
	socket, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   tracker.IP,
		Port: tracker.Port,
	})
	localIPStr := strings.Split(socket.LocalAddr().String(), ":")
	localPort, _ := strconv.Atoi(localIPStr[len(localIPStr)-1])
	if err != nil {
		fmt.Printf("%v announce dial error: %v\n", tracker.Host, err)
		return
	}
	defer func() { _ = socket.Close() }()

	// IPv4 announce request:
	//
	// Offset  Size    Name    Value
	// 0       64-bit integer  connection_id
	// 8       32-bit integer  action          1 // announce
	// 12      32-bit integer  transaction_id
	// 16      20-byte string  info_hash
	// 36      20-byte string  peer_id
	// 56      64-bit integer  downloaded
	// 64      64-bit integer  left
	// 72      64-bit integer  uploaded
	// 80      32-bit integer  event           0 // 0: none; 1: completed; 2: started; 3: stopped
	// 84      32-bit integer  IP address      0 // default
	// 88      32-bit integer  key
	// 92      32-bit integer  num_want        -1 // default
	// 96      16-bit integer  port
	// 98
	key := 0x1a7e3d22
	numWant := -1
	payload := make([]byte, 98)
	binary.BigEndian.PutUint64(payload[0:8], connId)
	binary.BigEndian.PutUint32(payload[8:12], uint32(ActionAnnounce))
	binary.BigEndian.PutUint32(payload[12:16], uint32(genTransactionID()))
	copy(payload[16:36], tf.InfoSHA[:])
	copy(payload[36:56], peerId[:])
	binary.BigEndian.PutUint64(payload[56:64], 0)
	binary.BigEndian.PutUint64(payload[64:72], uint64(tf.FileLen))
	binary.BigEndian.PutUint64(payload[72:80], 0)
	binary.BigEndian.PutUint32(payload[80:84], 0)
	binary.BigEndian.PutUint32(payload[84:88], 0)
	binary.BigEndian.PutUint32(payload[88:92], uint32(key))
	binary.BigEndian.PutUint32(payload[92:96], uint32(numWant))
	binary.BigEndian.PutUint16(payload[96:98], uint16(localPort))
	_, err = socket.Write(payload)
	if err != nil {
		fmt.Printf("%v announce write payload error: %v\n", tracker.Host, err)
		return
	}
	// IPv4 announce response:
	//
	// 0           32-bit integer  action          1 // announce
	// 4           32-bit integer  transaction_id
	// 8           32-bit integer  interval
	// 12          32-bit integer  leechers
	// 16          32-bit integer  seeders
	// 20 + 6 * n  32-bit integer  IP address
	// 24 + 6 * n  16-bit integer  TCP port
	// 20 + 6 * N
	data := make([]byte, 1220) // 20 + 200 * 6
	_ = socket.SetReadDeadline(time.Now().Add(time.Duration(RetrievePeersTimeout) * time.Second))
	n, remoteAddr, err := socket.ReadFromUDP(data)
	if err != nil {
		fmt.Printf("%v announce read from udp error: %v\n", tracker.Host, err)
		return
	}
	if binary.BigEndian.Uint32(data[:4]) == ActionError {
		fmt.Printf("%v announce response error\n", tracker.Host)
		return
	}
	fmt.Printf("announce recv:%v addr:%v count:%v\n", data[:n], remoteAddr, n)

	// peers info
	peerNum := (len(data) - 20) / 6
	for i := 0; i < peerNum; i++ {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(data[20+6*i:24+6*i]))
		port := binary.BigEndian.Uint16(data[24+6*i : 26+6*i])
		if port == 0 {
			continue
		}
		peerChan <- &PeerInfo{
			Ip:   ip,
			Port: port,
		}
	}
}

func isHTTPTrackerUrl(url string) bool {
	return strings.HasPrefix(url, "http")
}

func isUDPTrackerUrl(url string) bool {
	return strings.HasPrefix(url, "udp")
}

func genTransactionID() int32 {
	return rand.Int31n(214748)
}
