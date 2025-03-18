package main

import (
	"bytes"
	"encoding/gob"
	"time"
)

var b bytes.Buffer

func encode(gl GameLog) ([]byte, error) {
	enc := gob.NewEncoder(&b)

	if err := enc.Encode(gl); err != nil {
		return []byte{}, err
	}

	return b.Bytes(), nil
}

func decode(data []byte) (GameLog, error) {
	var gl GameLog
	dec := gob.NewDecoder(&b)
	if err := dec.Decode(&gl); err != nil {
		return GameLog{}, err
	}

	return gl, nil
}

// don't touch below this line

type GameLog struct {
	CurrentTime time.Time
	Message     string
	Username    string
}
