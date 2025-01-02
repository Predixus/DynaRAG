package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
)

// calculateMetadataHash calculate a hash from an arbitrary object
func CalculateMetadataHash(metadata map[string]interface{}) (string, error) {
	jsonBytes, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	hash := md5.Sum(jsonBytes)
	return hex.EncodeToString(hash[:]), nil
}
