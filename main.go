package main

import (
	"fmt"
	"log"
	"os"

	"github.com/JM-Monteiro/torrent-client/torrentfile"
	"github.com/anacrolix/torrent/metainfo"
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

	_, err := torrentfile.Open("originalTest.torrent")
	//_, err := torrentfile.Open("test1.torrent")
	if err != nil {
		log.Fatal(err)
	}

	mi, err := metainfo.LoadFromFile("originalTest.torrent")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(mi.HashInfoBytes())

	/*err = tf.DownloadToFile("real")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(tf.AnnounceList)*/

}
