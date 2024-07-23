package controller

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
)

func md5hash(str string) string {
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

func validateJson(jsonContent string) error {
	var myJon json.RawMessage
	return json.Unmarshal([]byte(jsonContent), &myJon)
}
