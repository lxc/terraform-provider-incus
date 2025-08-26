package common

import "encoding/json"

func ToMapStringAnySlice(in any) ([]map[string]any, error) {
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	value := []map[string]any{}

	err = json.Unmarshal(data, &value)
	return value, err
}

func ToMapStringAny(in any) (map[string]any, error) {
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	value := map[string]any{}

	err = json.Unmarshal(data, &value)
	return value, err
}
