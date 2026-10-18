package update

import (
	"encoding/json"
	"saveme/internal/core"
)

func decodeStatus(data []byte) (core.Response, error) {
	var result core.Response
	err := json.Unmarshal(data, &result)
	return result, err
}
