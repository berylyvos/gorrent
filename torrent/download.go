package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"runtime"
	"time"
)

type TorrentTask struct {
	PeerId   [PeerIdLen]byte
	PeerMap  map[string]*PeerInfo
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

const MaxBlockSize = 16384 // 16KB
const MaxBacklog = 5

func (state *taskState) handleMsg() error {
	msg, err := state.conn.ReadMsg()
	if err != nil {
		return err
	}
	// handle keep-alive
	if msg == nil {
		return nil
	}

	switch msg.Id {
	case MsgChoke:
		state.conn.Choked = true
	case MsgUnchoke:
		state.conn.Choked = false
	case MsgHave:
		index, err := GetHaveIndex(msg)
		if err != nil {
			return err
		}
		state.conn.BitField.SetPiece(index)
	case MsgPiece:
		n, err := CopyPieceData(state.index, state.data, msg)
		if err != nil {
			return err
		}
		state.downloaded += n
		state.backlog--
	}

	return nil
}

func downloadPiece(conn *PeerConn, task *pieceTask) (*pieceResult, error) {
	state := &taskState{
		index: task.index,
		conn:  conn,
		data:  make([]byte, task.length),
	}
	conn.SetDeadline(time.Now().Add(15 * time.Second))
	defer conn.SetDeadline(time.Time{})

	for state.downloaded < task.length {
		// If remote peer unchoked us, send requests until we have enough unfulfilled requests
		if !conn.Choked {
			for state.backlog < MaxBacklog && state.requested < task.length {
				blockSize := MaxBlockSize
				// Last block might be shorter than the typical block
				if task.length-state.requested < blockSize {
					blockSize = task.length - state.requested
				}
				msg := NewRequestMsg(state.index, state.requested, blockSize)
				_, err := state.conn.WriteMsg(msg)
				if err != nil {
					return nil, err
				}
				state.backlog++
				state.requested += blockSize
			}
		}
		err := state.handleMsg()
		if err != nil {
			return nil, err
		}
	}

	return &pieceResult{state.index, state.data}, nil
}

func checkPieceIntegrity(task *pieceTask, res *pieceResult) bool {
	sha := sha1.Sum(res.data)
	if !bytes.Equal(task.sha1[:], sha[:]) {
		fmt.Printf("check integrity failed, index: %v\n", res.index)
		return false
	}
	return true
}

func (t *TorrentTask) peerRoutine(peer *PeerInfo, taskQueue chan *pieceTask, resultQueue chan *pieceResult) {
	// set up conn with peer
	peerConn, err := NewConn(peer, t.InfoSHA, t.PeerId)
	if err != nil {
		fmt.Printf("failed to connect peer: %s:%d\n", peer.Ip.String(), peer.Port)
		return
	}
	defer peerConn.Close()

	fmt.Printf("complete handshake with peer: %s:%d\n", peer.Ip.String(), peer.Port)
	peerConn.WriteMsg(&PeerMsg{MsgInterested, nil})

	// retrieve piece tasks from task channel and try to download
	for task := range taskQueue {
		if !peerConn.BitField.HasPiece(task.index) {
			// if peer don't have current piece, put task back on task channel and continue
			taskQueue <- task
			continue
		}
		res, err := downloadPiece(peerConn, task)
		if err != nil {
			// if (network) error occurs while downloading piece, put task back and return
			// need to close the connection and kill this goroutine
			taskQueue <- task
			fmt.Println("failed to down piece: " + err.Error())
			return
		}
		if !checkPieceIntegrity(task, res) {
			// if piece integrity check fails, put cur task back on task channel and continue to handle next task
			taskQueue <- task
			continue
		}
		// successfully downloaded and checked, send to result channel
		resultQueue <- res
	}
}

func (t *TorrentTask) getPieceBounds(index int) (begin, end int) {
	begin = index * t.PieceLen
	end = begin + t.PieceLen
	if end > t.FileLen {
		end = t.FileLen
	}
	return
}

func (t *TorrentTask) Download() ([]byte, error) {
	fmt.Println("start downloading " + t.FileName)
	// split pieceTasks and init task & result channel
	pieceCount := len(t.PieceSHA)
	taskQueue := make(chan *pieceTask, pieceCount)
	resultQueue := make(chan *pieceResult)
	for idx, sha := range t.PieceSHA {
		begin, end := t.getPieceBounds(idx)
		taskQueue <- &pieceTask{
			index:  idx,
			sha1:   sha,
			length: end - begin,
		}
	}
	// init goroutines for each peer
	for _, peer := range t.PeerMap {
		go t.peerRoutine(peer, taskQueue, resultQueue)
	}
	// collect piece result
	buf := make([]byte, t.FileLen)
	count := 0
	for count < pieceCount {
		res := <-resultQueue
		begin, end := t.getPieceBounds(res.index)
		copy(buf[begin:end], res.data)
		count++
		// print progress
		percent := float64(count) / float64(pieceCount) * 100
		numWorkers := runtime.NumGoroutine() - 1 // subtract 1 for main thread
		fmt.Printf("downloaded piece #%d from %d peers in progress: (%0.2f%%)\n", res.index, numWorkers, percent)
	}
	close(taskQueue)
	close(resultQueue)

	return buf, nil
}
