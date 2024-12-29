/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/
package utils

import (
	"encoding/json"
	"fmt"
)

func ToJSON(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%q", string(b)), nil
}