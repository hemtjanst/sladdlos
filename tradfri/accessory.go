package tradfri

import (
	"encoding/json"
	"image/color"
	"log"
	"strconv"
	"time"
)

type Accessory struct {
	observable
	pendingChanges *Accessory
	BaseType
	Type       DeviceType  `json:"5750,omitempty"`
	DeviceInfo *DeviceInfo `json:"3,omitempty"`
	Alive      YesNo       `json:"9019,omitempty"`
	LastSeen   int64       `json:"9020,omitempty"`
	Lights     []*Light    `json:"3311,omitempty"`
	Plugs      []*Plug     `json:"3312,omitempty"`
	Sensors    []*Sensor   `json:"3300,omitempty"`
	Switches   []*Switch   `json:"15009,omitempty"`
	OTAUpdate  YesNo       `json:"9054,omitempty"`
}

func (a *Accessory) IsLight() bool {
	return a.Type == TypeLight && len(a.Lights) > 0
}

func (a *Accessory) IsRemote() bool {
	return a.Type == TypeRemote
}

func (a *Accessory) IsPlug() bool {
	return a.Type == TypePlug
}

func (a *Accessory) Plug() *Plug {
	if len(a.Plugs) > 0 {
		return a.Plugs[0]
	}
	return nil
}

func (a *Accessory) Light() *Light {
	if len(a.Lights) > 0 {
		return a.Lights[0]
	}
	return nil
}

func (a *Accessory) IsAlive() bool {
	return a.Alive == Yes
}

func (a *Accessory) LastSeenTime() time.Time {
	return time.Unix(a.LastSeen, 0)
}

func (a *Accessory) update(cb func(ch *Accessory)) {
	a.Lock()
	defer a.Unlock()
	if a.pendingChanges == nil {
		a.pendingChanges = &Accessory{
			BaseType: BaseType{tree: a.tree},
		}
		time.AfterFunc(50*time.Millisecond, func() {
			a.Lock()
			defer a.Unlock()
			a.pendingChanges.Lock()
			defer a.pendingChanges.Unlock()
			b, err := json.Marshal(a.pendingChanges)
			a.pendingChanges = nil
			if err != nil {
				log.Printf("Error marshaling pending changes json: %v", err)
				return
			}
			url := "15001/" + strconv.Itoa(a.GetInstanceID())
			log.Printf("Sending to %s: %s", url, string(b))
			if err := a.tree.transport.Put(url, b); err != nil {
				log.Printf("Error sending data %s: %v", string(b), err)
			}
		})
	}
	a.pendingChanges.Lock()
	defer a.pendingChanges.Unlock()
	cb(a.pendingChanges)
}

func (a *Accessory) updateDimmable(cb func(ch *Dimmable)) {
	a.update(func(ch *Accessory) {
		var l *Dimmable
		if ch.IsLight() {
			l = &ch.Light().Dimmable
		} else {
			li := &Light{}
			ch.Lights = []*Light{li}
			l = &li.Dimmable
		}
		cb(l)
	})
}

func (a *Accessory) updateOnOff(cb func(ch *OnOff)) {
	a.update(func(ch *Accessory) {
		var l *OnOff
		if ch.IsLight() {
			l = &ch.Light().OnOff
		} else if ch.IsPlug() {
			l = &ch.Plug().OnOff
		} else {
			li := &Light{}
			ch.Lights = []*Light{li}
			l = &li.OnOff
		}
		cb(l)
	})
}

func (a *Accessory) updateLight(cb func(ch *Light)) {
	a.update(func(ch *Accessory) {
		l := ch.Light()
		if l == nil {
			l = &Light{}
			ch.Lights = []*Light{l}
		}
		cb(l)
	})
}

func (a *Accessory) SetOn(on bool) {
	if !a.IsLight() {
		return
	}
	newVal := ToYesNo(on)
	a.updateOnOff(func(ch *OnOff) {
		ch.On = &newVal
	})
}

func (a *Accessory) SetDim(dim int) {
	if !a.IsLight() {
		return
	}
	newVal := calcDim(dim)
	a.updateDimmable(func(ch *Dimmable) {
		ch.Dim = &newVal
	})
}

func (a *Accessory) SetName(name string) {
	a.update(func(ch *Accessory) {
		ch.Name = name
	})
}

func (a *Accessory) SetColor(c color.Color) {
	if !a.IsLight() {
		return
	}
	a.updateLight(func(ch *Light) {
		ch.SetColor(c)
	})
}

func (a *Accessory) SetColorTemp(c string) {
	if !a.IsLight() {
		return
	}
	a.updateLight(func(ch *Light) {
		ch.SetColorTemp(c)
	})
}

func (a *Accessory) SetColorCold() {
	a.SetColorTemp(Cold)
}

func (a *Accessory) SetColorNormal() {
	a.SetColorTemp(Normal)
}

func (a *Accessory) SetColorWarm() {
	a.SetColorTemp(Warm)
}
