package strapi

import "encoding/json"

type response struct {
	Data  json.RawMessage `json:"data"`
	Meta  map[string]any  `json:"meta"`
	Error *errorPayload   `json:"error"`
}

type errorPayload struct {
	Status  int            `json:"status"`
	Name    string         `json:"name"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

type documentDTO struct {
	ID         int            `json:"id"`
	DocumentID string         `json:"documentId"`
	Meta       map[string]any `json:"meta"`
	Fields     map[string]any `json:"-"`
}

func (d *documentDTO) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if value, ok := raw["id"]; ok {
		if err := json.Unmarshal(value, &d.ID); err != nil {
			return err
		}
		delete(raw, "id")
	}
	if value, ok := raw["documentId"]; ok {
		if err := json.Unmarshal(value, &d.DocumentID); err != nil {
			return err
		}
		delete(raw, "documentId")
	}
	if value, ok := raw["meta"]; ok {
		if err := json.Unmarshal(value, &d.Meta); err != nil {
			return err
		}
		delete(raw, "meta")
	}
	d.Fields = make(map[string]any, len(raw))
	for key, value := range raw {
		var decoded any
		if err := json.Unmarshal(value, &decoded); err != nil {
			return err
		}
		d.Fields[key] = decoded
	}
	return nil
}
