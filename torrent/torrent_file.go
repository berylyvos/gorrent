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
	"strings"
)

type file struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
}

type rawInfo struct {
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
	PieceLength int    `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
}

type rawInfoMulti struct {
	Files []file `bencode:"files"`
	rawInfo
}

type rawFile struct {
	Announce     string       `bencode:"announce"`
	AnnounceList [][]string   `bencode:"announce-list"`
	Info         rawInfo      `bencode:"info"`
	InfoMulti    rawInfoMulti `bencode:"info"`
}

const ShaLen int = 20

type File struct {
	Length int
	Path   string
}

type TorrentFile struct {
	Announce     string
	AnnounceList []string
	InfoSHA      [ShaLen]byte
	FileList     []File
	FileName     string
	FileLen      int
	PieceLen     int
	PieceSHA     [][ShaLen]byte
	HasMulti     bool
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

	tf := newTorrentFile(raw)
	setInfoSha(raw, tf)
	setPieceSha(raw, tf)
	setFileLen(tf)

	return tf, nil
}

func (tf *TorrentFile) BuildTorrentTask(path string) (*TorrentTask, error) {
	// generate random peerId
	var peerId [PeerIdLen]byte
	_, _ = rand.Read(peerId[:])

	// retrieve peers from tracker
	peers := RetrievePeers(tf, peerId)
	if len(peers) == 0 {
		return nil, fmt.Errorf("there is no peers")
	}

	return &TorrentTask{
		PeerId:   peerId,
		PeerList: peers,
		InfoSHA:  tf.InfoSHA,
		FileName: tf.FileName,
		FileLen:  tf.FileLen,
		PieceLen: tf.PieceLen,
		PieceSHA: tf.PieceSHA,
	}, nil
}

func (tf *TorrentFile) DownloadToFile(path string) error {
	// build torrent task
	task, err := tf.BuildTorrentTask(path)
	if err != nil {
		return fmt.Errorf("build torrent task error: %v", err.Error())
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

func flattenFiles(files []file) []File {
	if files == nil {
		return nil
	}
	res := make([]File, len(files))
	for i, f := range files {
		res[i] = File{
			Path:   strings.Join(f.Path, "/"),
			Length: f.Length,
		}
	}
	return res
}

func flattenAnnounceList(list [][]string) []string {
	if list == nil {
		return nil
	}
	// shape of list can be N x 1 or 1 x N
	var res []string
	for _, lst := range list {
		for _, x := range lst {
			res = append(res, x)
		}
	}
	return res
}

func newTorrentFile(raw *rawFile) *TorrentFile {
	tf := new(TorrentFile)
	tf.Announce = raw.Announce
	tf.AnnounceList = flattenAnnounceList(raw.AnnounceList)
	tf.FileList = flattenFiles(raw.InfoMulti.Files)
	if tf.FileList != nil {
		tf.HasMulti = true
	}
	tf.FileName = raw.Info.Name
	tf.FileLen = raw.Info.Length
	tf.PieceLen = raw.Info.PieceLength
	return tf
}

// setInfoSha compute InfoSHA which is the SHA-1 hash of the entire bencoded info dict
// Be careful! If there's only a single file, bencoded data should not contain `files`.
func setInfoSha(raw *rawFile, tf *TorrentFile) {
	buf := new(bytes.Buffer)
	wLen := 0
	if tf.HasMulti {
		wLen = bencode.Marshal(buf, raw.InfoMulti)
	} else {
		wLen = bencode.Marshal(buf, raw.Info)
	}
	if wLen == 0 {
		fmt.Println("raw file info marshal error")
	}
	tf.InfoSHA = sha1.Sum(buf.Bytes())
}

// setPieceSha compute PieceSHA which is a slice of each piece's SHA-1
// raw.Info.Pieces is a big binary blob containing the SHA-1 hashes of
// each piece, now we want to split it into pieces.
func setPieceSha(raw *rawFile, tf *TorrentFile) {
	piecesBytes := []byte(raw.Info.Pieces)
	piecesCnt := len(piecesBytes) / ShaLen
	pieceSHA := make([][ShaLen]byte, piecesCnt)
	for i := 0; i < piecesCnt; i++ {
		copy(pieceSHA[i][:], piecesBytes[i*ShaLen:(i+1)*ShaLen])
	}
	tf.PieceSHA = pieceSHA
}

// setFileLen set total length of tf.FileList to tf.FileLen if tf.FileLen == 0
func setFileLen(tf *TorrentFile) {
	if tf.FileLen != 0 {
		return
	}
	fileLen := 0
	for _, f := range tf.FileList {
		fileLen += f.Length
	}
	tf.FileLen = fileLen
}
