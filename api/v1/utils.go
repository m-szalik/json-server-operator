package v1

import (
	"encoding/json"
)

func validateJson(jsonContent string) error {
	var myJon json.RawMessage
	return json.Unmarshal([]byte(jsonContent), &myJon)
}
