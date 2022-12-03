package main

import (
	"github.com/berylyvos/gorrent/torrent"
	"log"
)

func main() {
	inPath := "./testfile/The.Breakfast.Club.1985.REMASTERED.720p.BluRay.999MB.HQ.x265.10bit-GalaxyRG.torrent"
	outPath := "./nope"
	// open and parse torrent file
	tf, err := torrent.Open(inPath)
	if err != nil {
		log.Fatal(err)
	}
	// download and save
	err = tf.DownloadToFile(outPath)
	if err != nil {
		log.Fatal(err)
	}
}
