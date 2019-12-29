package configfile

import (
	"encoding/json"
	"os"
)

func ReadConfiguration(filename string, v interface{}) bool {
	file, err := os.Open(filename)
	if err == nil {
		defer file.Close()
		decoder := json.NewDecoder(file)
		err = decoder.Decode(v)
		if err == nil {
			return true
		}
	}
	return false
}

func WriteConfiguration(filename string, v interface{}) bool {
	return true
}
