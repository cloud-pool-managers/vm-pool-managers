package models

import "encoding/json"

type JSONStringSlice []string
type JSONStringMap map[string]string

func (j *JSONStringSlice) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONStringSlice) Value() (interface{}, error) {
	return json.Marshal(j)
}

func (j *JSONStringMap) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONStringMap) Value() (interface{}, error) {
	return json.Marshal(j)
}
