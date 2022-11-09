package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"github.com/berylyvos/gorrent/torrent"
	"os"
)

func main() {
	// parse torrent file
	file, err := os.Open("./testfile/debian-iso.torrent")
	if err != nil {
		fmt.Println("open file error", err)
		return
	}
	defer file.Close()
	tf, err := torrent.ParseFile(bufio.NewReader(file))
	if err != nil {
		fmt.Println("parse file error", err)
	}

	// generate random peerId
	var peerId [torrent.PeerIdLen]byte
	_, _ = rand.Read(peerId[:])
	// retrieve peers from tracker
	peers := torrent.RetrievePeers(tf, peerId)
	if len(peers) == 0 {
		fmt.Println("there is no peers")
		return
	}

	// build torrent task
	task := &torrent.TorrentTask{
		PeerId:   peerId,
		PeerList: peers,
		InfoSHA:  tf.InfoSHA,
		FileName: tf.FileName,
		FileLen:  tf.FileLen,
		PieceLen: tf.PieceLen,
		PieceSHA: tf.PieceSHA,
	}
	// download from peers
	err = torrent.Download(task)
	if err != nil {
		fmt.Println("download error", err)
		return
	}
}
