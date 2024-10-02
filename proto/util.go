package proto

import (
	"bytes"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io/ioutil"
	"strings"
)

func Utf8ToGbk(text []byte) string {

	r := bytes.NewReader(text)

	decoder := transform.NewReader(r, simplifiedchinese.GBK.NewDecoder()) //GB18030

	content, _ := ioutil.ReadAll(decoder)

	return strings.ReplaceAll(string(content), string([]byte{0x00}), "")
}
