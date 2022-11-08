package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
)

type TorrentTask struct {
	PeerId   [PeerIdLen]byte
	PeerList []PeerInfo
	InfoSHA  [ShaLen]byte
	FileName string
	FileLen  int
	PieceLen int
	PieceSHA [][ShaLen]byte
}

type pieceTask struct {
	index  int
	sha1   [ShaLen]byte
	length int
}

type taskState struct {
	index      int
	conn       *PeerConn
	requested  int
	downloaded int
	backlog    int
	data       []byte
}

type pieceResult struct {
	index int
	data  []byte
}

const BlockSize = 16384 // 16KB
const MaxBacklog = 5

func (state *taskState) handleMsg() error {
	// TODO

	return nil
}

func downloadPiece(conn *PeerConn, task *pieceTask) (*pieceResult, error) {
	// TODO

	return nil, nil
}

func checkPiece(task *pieceTask, res *pieceResult) bool {
	sha := sha1.Sum(res.data)
	if !bytes.Equal(task.sha1[:], sha[:]) {
		fmt.Printf("check integrity failed, index: %v\n", res.index)
		return false
	}
	return true
}

func (t *TorrentTask) peerRoutine(peer PeerInfo, taskQueue chan *pieceTask, resultQueue chan *pieceResult) {
	// TODO
}

func (t *TorrentTask) getPieceBounds(index int) (begin, end int) {
	begin = index * t.PieceLen
	end = begin + t.PieceLen
	if end > t.FileLen {
		end = t.FileLen
	}
	return
}

func Download(task *TorrentTask) error {
	fmt.Println("start downloading " + task.FileName)
	// split pieceTasks and init task & result channel
	pieceCount := len(task.PieceSHA)
	taskQueue := make(chan *pieceTask, pieceCount)
	resultQueue := make(chan *pieceResult)
	for idx, sha := range task.PieceSHA {
		begin, end := task.getPieceBounds(idx)
		taskQueue <- &pieceTask{
			index:  idx,
			sha1:   sha,
			length: end - begin,
		}
	}
	// init goroutines for each peer
	for _, peer := range task.PeerList {
		go task.peerRoutine(peer, taskQueue, resultQueue)
	}
	// collect piece result
	buf := make([]byte, task.FileLen)
	count := 0
	for count < pieceCount {
		res := <-resultQueue
		begin, end := task.getPieceBounds(res.index)
		copy(buf[begin:end], res.data)
		count++
		// print progress
		percent := float64(count) / float64(pieceCount) * 100
		fmt.Printf("downloading, progress: (%0.2f%%)\n", percent)
	}
	close(taskQueue)
	close(resultQueue)

	// save data to file
	file, err := os.Create(task.FileName)
	if err != nil {
		fmt.Println("fail to create file: " + task.FileName)
		return err
	}
	_, err = file.Write(buf)
	if err != nil {
		fmt.Println("fail to save data")
		return err
	}
	return nil
}
