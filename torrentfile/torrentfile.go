package torrentfile

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/JM-Monteiro/torrent-client/p2p"
	"github.com/JM-Monteiro/torrent-client/peers"

	bencode2 "github.com/anacrolix/torrent/bencode"
	"github.com/jackpal/bencode-go"
)

// Port to listen on
const Port uint16 = 6881

// TorrentFile encodes the metadata from a .torrent file
type TorrentFile struct {
	Announce     string
	AnnounceList []string
	InfoHash     [20]byte
	PieceHashes  [][20]byte
	PieceLength  int
	Length       int
	Name         string
	Files        []string
	FileLenght   map[string]int
}

type Files struct {
	Lenght int      `bencode:"length"`
	Path   []string `bencode:"path"`
}

type bencodeInfo struct {
	Pieces      string  `bencode:"pieces"`
	PieceLength int     `bencode:"piece length"`
	Length      int     `bencode:"length"`
	Name        string  `bencode:"name"`
	Files       []Files `bencode:"files"`
}

type bencodeTorrent struct {
	AnnounceList [][]string  `bencode:"announce-list"`
	Announce     string      `bencode:"announce"`
	Info         bencodeInfo `bencode:"info"`
	InfoBytes    []byte
}

type MetaInfo struct {
	InfoBytes bencode2.Bytes `bencode:"info,omitempty"`
}

// DownloadToFile downloads a torrent and writes it to a file
func (t *TorrentFile) DownloadToFile(path string) error {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])

	if err != nil {
		return err
	}

	log.Println("LAUNCHING HTTP")

	peers := make([]peers.Peer, 0)

	peers, err = t.requestPeers(peerID, Port)
	if err != nil {
		log.Println("TRACKER NOT GOOD", err)
		return err
	}
	/*
		log.Println("LAUNCHING DHT")

		dhtPeers, err := p2p.GetPeersFromDHT(t.InfoHash, int(Port)+1)

		if err != nil {
			return err
		}

		peers = append(peers, dhtPeers...)*/

	torrent := p2p.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		Name:        t.Name,
	}

	recieverData := make(chan []byte)
	recieverPieceId := make(chan int)

	go t.recievePieces(path, recieverData, recieverPieceId)

	_, err = torrent.Download(recieverData, recieverPieceId)
	if err != nil {
		return err
	}

	return nil
}

func (t *TorrentFile) recievePieces(path string, data chan []byte, id chan int) error {
	fullPath := path
	donePieces := 0
	if len(t.Files) > 0 {
		err := os.Mkdir(path, 0755)
		if err != nil {
			return err
		}
		fullPath = path + "/" + t.Name
		err = os.Mkdir(fullPath, 0755)
		if err != nil {
			return err
		}
	}

	fileArray := make([]*os.File, 0)
	pieceMap := make(map[int]*os.File)
	startPiece := make(map[*os.File]int)
	for _, file := range t.Files {
		outFile, err := os.Create(fullPath + "/" + file)
		if err != nil {
			return nil
		}
		fileArray = append(fileArray, outFile)
	}

	curLenght := 0
	curFile := 0
	for i, _ := range t.PieceHashes {
		pieceMap[i] = fileArray[curFile]
		curLenght = curLenght + t.PieceLength
		if curLenght >= t.FileLenght[t.Files[curFile]] && curFile+1 < len(fileArray) {
			curFile = curFile + 1
			curLenght = 0
			startPiece[fileArray[curFile]] = i + 1
		}
	}

	fileBuffer := make(map[int][]byte, len(t.Files))
	for i := range fileBuffer {
		buf := make([]byte, t.FileLenght[t.Files[i]])
		fileBuffer[i] = buf
	}

	for _, file := range fileArray {
		defer file.Close()
	}

	for donePieces < len(t.PieceHashes) {
		newData := <-data
		newId := <-id

		file := pieceMap[newId]
		startP := startPiece[file]

		index := newId - startP

		file.WriteAt(newData, int64(index)*int64(t.PieceLength))

		donePieces = donePieces + 1

		log.Println("Wrote File: ", file.Name(), "at pos: ", index)
	}

	return nil

}

func getInfoBytes(file io.Reader, bto *bencodeTorrent) error {

	var mi MetaInfo
	d := bencode2.NewDecoder(file)
	err := d.Decode(&mi)
	if err != nil {
		return err
	}

	bto.InfoBytes = mi.InfoBytes

	return nil
}

// Open parses a torrent file
func Open(path string) (TorrentFile, error) {
	file, err := os.Open(path)

	if err != nil {
		return TorrentFile{}, err
	}

	defer file.Close()

	bto := bencodeTorrent{}

	err = bencode.Unmarshal(file, &bto)

	file.Seek(0, 0)

	getInfoBytes(file, &bto)

	if err != nil {
		return TorrentFile{}, err
	}

	if err != nil {
		return TorrentFile{}, err
	}

	return bto.toTorrentFile()
}

func (i *bencodeTorrent) hash() ([20]byte, error) {

	h := sha1.Sum(i.InfoBytes)
	return h, nil
}

func (i *bencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer

	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [20]byte{}, err
	}

	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func (i *bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	hashLen := 20 // Length of SHA-1 hash
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		err := fmt.Errorf("Received malformed pieces of length %d", len(buf))
		return nil, err
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}

func (bto *bencodeTorrent) toTorrentFile() (TorrentFile, error) {
	infoHash, err := bto.hash()
	fmt.Println(hex.EncodeToString(infoHash[:]))
	if err != nil {
		return TorrentFile{}, err
	}
	pieceHashes, err := bto.Info.splitPieceHashes()
	if err != nil {
		return TorrentFile{}, err
	}

	files := make([]string, 0)
	fileSize := make(map[string]int)

	if len(bto.Info.Files) > 0 {
		for _, f := range bto.Info.Files {
			filePath := ""
			for _, p := range f.Path {
				filePath = filePath + p
			}
			files = append(files, filePath)
			fileSize[filePath] = f.Lenght
		}
	}

	addrList := make([]string, 0)
	for _, annouce := range bto.AnnounceList {
		addr := ""
		for _, char := range annouce {
			addr = addr + char
		}
		addrList = append(addrList, addr)
	}

	if len(addrList) == 0 {
		addrList = nil
	}
	if len(files) == 0 {
		files = append(files, bto.Info.Name)
	} else {
		for _, v := range fileSize {
			bto.Info.Length = bto.Info.Length + v
		}
	}

	t := TorrentFile{
		Announce:     bto.Announce,
		AnnounceList: addrList,
		InfoHash:     infoHash,
		PieceHashes:  pieceHashes,
		PieceLength:  bto.Info.PieceLength,
		Length:       bto.Info.Length,
		Name:         bto.Info.Name,
		Files:        files,
		FileLenght:   fileSize,
	}
	return t, nil
}
