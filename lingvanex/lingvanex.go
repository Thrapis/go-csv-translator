package lingvanex

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

const LINGVANEX_ADDRESS = "127.0.0.1"
const LINGVANEX_PORT = 8000

func Translate(text, from, to string) (string, error) {
	url := fmt.Sprintf("http://%s:%d", LINGVANEX_ADDRESS, LINGVANEX_PORT)

	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte(text)))
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	return string(body), nil
}
