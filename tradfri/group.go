package tradfri

import (
	"encoding/json"
	"log"
	"strconv"
	"time"
)

type Group struct {
	observable
	pendingChanges *Group
	BaseType
	Dimmable
	Scene      *int  `json:"9039,omitempty"`
	Members    []int `json:"9018,omitempty"`
	memberRefs []*Accessory
	Scenes     map[int]*Scene `json:"-"`
}

type grpAccessoryRef struct {
	DeviceIDs []int `json:"9003,omitempty"`
}
type grpMemberRef struct {
	Accessory *grpAccessoryRef `json:"15002,omitempty"`
}

func (g *Group) UnmarshalJSON(b []byte) error {
	type Alias Group
	aux := &struct {
		Members *grpMemberRef `json:"9018,omitempty"`
		*Alias
	}{Alias: (*Alias)(g)}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if aux.Members == nil || aux.Members.Accessory == nil || aux.Members.Accessory.DeviceIDs == nil {
		return nil
	}

	g.Members = aux.Members.Accessory.DeviceIDs
	return nil
}

func (g *Group) MarshalJSON() ([]byte, error) {
	type Alias Group
	aux := &struct {
		*Alias
		Members *grpMemberRef `json:"9018,omitempty"`
	}{
		Alias: (*Alias)(g),
	}

	if g.Members != nil {
		aux.Members = &grpMemberRef{&grpAccessoryRef{g.Members}}
	} else {
		aux.Members = nil
	}

	return json.Marshal(aux)
}

func (g *Group) update(cb func(ch *Group)) {
	g.Lock()
	defer g.Unlock()
	if g.pendingChanges == nil {
		g.pendingChanges = &Group{
			BaseType: BaseType{tree: g.tree},
		}
		time.AfterFunc(50*time.Millisecond, func() {
			g.Lock()
			defer g.Unlock()
			g.pendingChanges.Lock()
			defer g.pendingChanges.Unlock()
			b, err := json.Marshal(g.pendingChanges)
			g.pendingChanges = nil
			if err != nil {
				log.Printf("Error marshaling pending changes json: %v", err)
				return
			}
			url := "15004/" + strconv.Itoa(g.GetInstanceID())
			log.Printf("Sending to %s: %s", url, string(b))
			if err := g.tree.transport.Put(url, b); err != nil {
				log.Printf("Error sending data %s: %v", string(b), err)
			}
		})
	}
	g.pendingChanges.Lock()
	defer g.pendingChanges.Unlock()
	cb(g.pendingChanges)
}

func (g *Group) SetOn(on bool) {
	newVal := ToYesNo(on)
	g.update(func(ch *Group) {
		ch.On = &newVal
	})
}

func (g *Group) SetDim(dim int) {
	newDim := calcDim(dim)
	g.update(func(ch *Group) {
		ch.Dim = &newDim
	})
}

func (g *Group) SetName(name string) {
	g.update(func(ch *Group) {
		ch.Name = name
	})
}
