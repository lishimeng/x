package hardware

import (
	"os"
	"regexp"
)

func saveFile(content string, file string) (err error) {
	f, err := os.OpenFile(file, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()
	_, _ = f.WriteString(content)
	return
}

func loadFile(file string) (content string, err error) {
	bs, err := os.ReadFile(file)
	if err != nil {
		return
	}
	content = string(bs)
	return
}

func isValidText(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9]`)
	return reg.ReplaceAllString(s, "")
}
