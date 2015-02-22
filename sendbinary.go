package websocks

import (
	"bytes"
	"image/png"
	"os"
)

// BinaryMsg struct is used to send binary files (ie. images and sounds)
// trough websocket to the front end.
type BinaryMsg struct {
	Id     string `json:"id"`
	Type   string `json:"type"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Data   []byte `json:"data"`
}

// LoadPng takes a filename, and returns *BinaryMsg or error.
func LoadPng(fileName string) (*BinaryMsg, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	m, err := png.Decode(file)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, m); err != nil {
		return nil, err
	}

	binMsg := &BinaryMsg{
		Id:     fileName,
		Type:   "png",
		Width:  m.Bounds().Max.X,
		Height: m.Bounds().Max.Y,
		Data:   buf.Bytes(),
	}
	return binMsg, nil
}
