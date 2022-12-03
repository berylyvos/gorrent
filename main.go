package main

import (
	"github.com/berylyvos/gorrent/torrent"
	"log"
)

func main() {
	inPath := "./testfile/cyberpunk.torrent"
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
