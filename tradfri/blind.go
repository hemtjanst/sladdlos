package tradfri

import (
	"encoding/json"
)

type Blind struct {
	BaseType
	Position *int `json:"5536,omitempty"`
}

func (b *Blind) Pos() int {
	if b.Position != nil {
		return *b.Position
	}
	return 0
}

func (b *Blind) UnmarshalJSON(data []byte) error {
	v := &struct {
		BaseType
		Position *float64 `json:"5536,omitempty"`
	}{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	b.BaseType = v.BaseType
	var pos int
	if v.Position != nil {
		pos = int(*v.Position)
		b.Position = &pos
	}
	return nil
}
