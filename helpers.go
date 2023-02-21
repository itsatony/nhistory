package nhistory

import (
	"crypto/md5"
	"errors"
	"strconv"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

const idAlphabet string = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_"

// NID generates a unique ID
// prefix: optional prefix. length of the prefix is added to the length of the ID, not substracted!
// length: length of the ID
func NID(prefix string, length int) (nid string) {
	var err error
	nid, err = gonanoid.Generate(idAlphabet, length)
	if err != nil {
		nid = strconv.FormatInt(time.Now().UnixMicro(), 10)
	}
	if len(prefix) > 0 {
		nid = prefix + "_" + nid
	}
	return nid
}

func HashIt(s string) string {
	b := []byte(s)
	md5 := md5.New()
	md5.Write(b)
	return string(md5.Sum(b))
}

func CreateRedisKey(KeyPartsArray []string, prefix string, keyseparator string) (string, error) {
	var redisKey = ""
	var allParts []string
	allParts = append(allParts, prefix)
	allParts = append(allParts, KeyPartsArray...)
	// nuts.L.Debugf("allparts : <%s>", allParts)
	for _, v := range allParts {
		if v == "" {
			err := errors.New("empty redis Key Part not allowed")
			return "", err
		}
	}
	redisKey = strings.Join(allParts, keyseparator)
	return redisKey, nil
}
