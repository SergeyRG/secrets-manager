package domain

import "crypto/rand"

type FileID string

type FileState byte

const (
	FileStateUnknown FileState = iota
	FileStateUploading
	FileStateReady
	FileStateMax
)

func (f FileState) ToString() string {
	str := "UNKNOWN"
	switch f {
	case FileStateUploading:
		str = "UPLOADING"
	case FileStateReady:
		str = "READY"
	}
	return str
}

type File struct {
	FileID    FileID
	FileState FileState
}

func NewFileID() FileID {
	return FileID(rand.Text())
}
