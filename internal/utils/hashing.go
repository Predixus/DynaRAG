package utils

import (
	"crypto/md5"
	"encoding/hex"
)

// calculateMetadataHash calculate a hash from an arbitrary object
func CalculateMetadataHash(jsonBytes []byte) (string, error) {
	hash := md5.Sum(jsonBytes)
	return hex.EncodeToString(hash[:]), nil
}
