package torrent

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseFile(t *testing.T) {
	tf, err := Open("../testfile/debian-iso.torrent")
	assert.Equal(t, nil, err)
	assert.Equal(t, "http://bttracker.debian.org:6969/announce", tf.Announce)
	assert.Equal(t, "debian-11.2.0-amd64-netinst.iso", tf.FileName)
	assert.Equal(t, 396361728, tf.FileLen) // 378 MB
	assert.Equal(t, 262144, tf.PieceLen)   // 256 KB
	assert.Equal(t, 396361728/262144, len(tf.PieceSHA))
	var expectHASH = [20]byte{0x28, 0xc5, 0x51, 0x96, 0xf5, 0x77, 0x53, 0xc4, 0xa,
		0xce, 0xb6, 0xfb, 0x58, 0x61, 0x7e, 0x69, 0x95, 0xa7, 0xed, 0xdb}
	assert.Equal(t, expectHASH, tf.InfoSHA)
}

func TestParseMultiFile(t *testing.T) {
	tf, err := Open("../testfile/The.Breakfast.Club.1985.REMASTERED.720p.BluRay.999MB.HQ.x265.10bit-GalaxyRG.torrent")
	fmt.Printf("%+v\n%v\n", tf.FileList, err)
	assert.Equal(t, nil, err)
	assert.Equal(t, "udp://tracker.coppersurfer.tk:6969/announce", tf.Announce)
	assert.Equal(t, "The.Breakfast.Club.1985.REMASTERED.720p.BluRay.999MB.HQ.x265.10bit-GalaxyRG[TGx]", tf.FileName)
	assert.Equal(t, 0, tf.FileLen)
	assert.Equal(t, 524288, tf.PieceLen) // 512 KB
}
