package proto

import (
	"bytes"
	"io/ioutil"
	"strings"
	"unicode"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func Utf8ToGbk(text []byte) string {

	r := bytes.NewReader(text)

	decoder := transform.NewReader(r, simplifiedchinese.GBK.NewDecoder()) //GB18030

	content, _ := ioutil.ReadAll(decoder)

	result := strings.ReplaceAll(string(content), string([]byte{0x00}), "")
	return strings.TrimFunc(result, func(r rune) bool {
		return unicode.IsControl(r)
	})
}
