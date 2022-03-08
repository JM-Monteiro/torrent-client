package main

import (
	"fmt"
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

func main() {

	tf, err := torrentfile.Open("originalTest.torrent")
	if err != nil {
		log.Fatal(err)
	}

	err = tf.DownloadToFile("testFiles")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(tf.AnnounceList)

}
