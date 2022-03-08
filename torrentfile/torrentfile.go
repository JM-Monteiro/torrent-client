package torrentfile

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/JM-Monteiro/torrent-client/p2p"
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
}

// DownloadToFile downloads a torrent and writes it to a file
func (t *TorrentFile) DownloadToFile(path string) error {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return err
	}

	peers, err := t.requestPeers(peerID, Port)
	if err != nil {
		return err
	}

	torrent := p2p.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		Name:        t.Name,
	}
	buf, err := torrent.Download()
	if err != nil {
		return err
	}

	if len(t.Files) > 0 {
		return t.writeMultipleFiles(path, buf)
	} else {
		outFile, err := os.Create(path)
		if err != nil {
			return err
		}

		defer outFile.Close()
		_, err = outFile.Write(buf)
		if err != nil {
			return err
		}
		return nil
	}
}

func (t *TorrentFile) writeMultipleFiles(path string, buf []byte) error {
	fullPath := path + "/" + t.Name
	err := os.Mkdir(fullPath, 0755)
	if err != nil {
		return err
	}

	bytesRead := 0
	for i, file := range t.Files {
		outFile, err := os.Create(fullPath + "/" + file)
		if err != nil {
			return err
		}
		defer outFile.Close()

		fileBuf := make([]byte, t.FileLenght[file])
		if i == 0 {
			fileBuf = append(fileBuf, buf[:t.FileLenght[file]]...)
		} else {
			fileBuf = append(fileBuf, buf[bytesRead+1:t.FileLenght[file]]...)
		}

		bytesRead = bytesRead + t.FileLenght[file]

		_, err = outFile.Write(fileBuf)
		if err != nil {
			return err
		}
	}
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
	if err != nil {
		return TorrentFile{}, err
	}

	return bto.toTorrentFile()
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
	infoHash, err := bto.Info.hash()
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

	/*if len(bto.Info.Files) > 0 {
		for _, f := range bto.Info.Files {
			filePath := ""
			for _, p := range f.Path {
				filePath = filePath + p
			}
			files = append(files, filePath)
			fileSize[filePath] = f.Lenght
		}
	}*/

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
		files = nil
		fileSize = nil
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
