package main

import (
	"bufio"
	"errors"
	"os"

	"github.com/gorilla/securecookie"
)

type AuthKeyFile struct {
	Path string
	keys [][]byte
}

func LoadAuthFile(path string, forceNewKey bool, keyLen int) (kf *AuthKeyFile, err error) {
	kf = new(AuthKeyFile)
	kf.Path = path

	var (
		file *os.File
	)

	file, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0600)
	if err != nil {
		kf = nil
		return
	}
	defer file.Close()

	scan := bufio.NewScanner(file)

	for scan.Scan() {
		kf.keys = append(kf.keys, scan.Bytes())
	}

	err = scan.Err()
	if err != nil {
		kf = nil
		return
	}

	// we need a new key
	if kf.keys == nil || forceNewKey {
		nk := securecookie.GenerateRandomKey(keyLen)
		if nk == nil {
			kf = nil
			err = errors.New("could not generate auth key")
			return
		}

		// save it
		_, err = file.Write(nk)
		if err != nil {
			kf = nil
			return
		}
		file.WriteString("\n")

		kf.keys = append(kf.keys, nk)
	}

	return
}

func (kf *AuthKeyFile) AsKeyPairs() (ret [][]byte) {
	ret = make([][]byte, len(kf.keys) * 2)

	for i, k := range kf.keys {
		ret[i * 2] = k
		// encrypt key
		ret[i * 2 + 1] = nil
	}

	return
}
