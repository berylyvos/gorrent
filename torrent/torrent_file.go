package torrent

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"github.com/berylyvos/gorrent/bencode"
	"io"
	"os"
)

type rawInfo struct {
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
	PieceLength int    `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
}

type rawFile struct {
	Announce string  `bencode:"announce"`
	Info     rawInfo `bencode:"info"`
}

const ShaLen int = 20

type TorrentFile struct {
	Announce string
	InfoSHA  [ShaLen]byte
	FileName string
	FileLen  int
	PieceLen int
	PieceSHA [][ShaLen]byte
}

func Open(path string) (*TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	tf, err := ParseFile(bufio.NewReader(file))
	if err != nil {
		return nil, err
	}
	return tf, nil
}

func ParseFile(r io.Reader) (*TorrentFile, error) {
	raw := new(rawFile)
	err := bencode.Unmarshal(r, raw)
	if err != nil {
		fmt.Println("failed to parse torrent file")
		return nil, err
	}
	tf := new(TorrentFile)
	tf.Announce = raw.Announce
	tf.FileName = raw.Info.Name
	tf.FileLen = raw.Info.Length
	tf.PieceLen = raw.Info.PieceLength

	// compute InfoSHA which is the SHA-1 hash of the entire bencoded info dict
	buf := new(bytes.Buffer)
	wLen := bencode.Marshal(buf, raw.Info)
	if wLen == 0 {
		fmt.Println("raw file info marshal error")
	}
	tf.InfoSHA = sha1.Sum(buf.Bytes())

	// raw.Info.Pieces is a big binary blob containing the SHA-1 hashes of each piece
	// now we want to split pieces into small piece
	// compute PieceSHA which is a slice of each piece's SHA-1 hash
	piecesBytes := []byte(raw.Info.Pieces)
	piecesCnt := len(piecesBytes) / ShaLen
	pieceSHA := make([][ShaLen]byte, piecesCnt)
	for i := 0; i < piecesCnt; i++ {
		copy(pieceSHA[i][:], piecesBytes[i*ShaLen:(i+1)*ShaLen])
	}
	tf.PieceSHA = pieceSHA
	return tf, nil
}

func (tf *TorrentFile) DownloadToFile(path string) error {
	// generate random peerId
	var peerId [PeerIdLen]byte
	_, _ = rand.Read(peerId[:])
	// retrieve peers from tracker
	peers := RetrievePeers(tf, peerId)
	if len(peers) == 0 {
		return fmt.Errorf("there is no peers")
	}

	// build torrent task
	task := &TorrentTask{
		PeerId:   peerId,
		PeerList: peers,
		InfoSHA:  tf.InfoSHA,
		FileName: tf.FileName,
		FileLen:  tf.FileLen,
		PieceLen: tf.PieceLen,
		PieceSHA: tf.PieceSHA,
	}
	// download from peers
	buf, err := task.Download()
	if err != nil {
		return fmt.Errorf("download error: %v", err.Error())
	}
	// save data to file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("fail to create file: " + task.FileName)
	}
	_, err = file.Write(buf)
	if err != nil {
		return fmt.Errorf("fail to save data to file: %v", err.Error())
	}
	return nil
}
