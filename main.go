package main

import (
	"log"
	"os"

	"github.com/JM-Monteiro/torrent-client/torrentfile"
)

func mainReal() {
	inPath := os.Args[1]
	outPath := os.Args[2]

	tf, err := torrentfile.Open(inPath)
	if err != nil {
		log.Fatal(err)
	}

	err = tf.DownloadToFile(outPath)
	if err != nil {
		log.Fatal(err)
	}
}
