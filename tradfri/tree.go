package tradfri

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type ObservableTransport interface {
	Transport
	Observe(method, uri string, callback func(data []byte)) error
}

type DiscoverCallback interface {
	OnNewAccessory(d *Accessory)
	OnNewGroup(g *Group)
	OnNewScene(g *Group, s *Scene)
}

type Transport interface {
	Get(uri string) ([]byte, error)
	Put(uri string, data []byte) error
	Delete(uri string) error
}

type Tree struct {
	sync.RWMutex
	Devices       map[int]*Accessory
	Groups        map[int]*Group
	Notifications []*Notification
	Gateway       *Gateway
	transport     Transport
	callback      []DiscoverCallback
}

func NewTree(transport Transport) *Tree {
	t := &Tree{
		Devices:       map[int]*Accessory{},
		Groups:        map[int]*Group{},
		Notifications: []*Notification{},
		Gateway:       &Gateway{},
		transport:     transport,
		callback:      []DiscoverCallback{},
	}
	t.Gateway.tree = t
	return t
}

func (t *Tree) AddCallback(callback DiscoverCallback) {
	t.callback = append(t.callback, callback)
}

func (t *Tree) Populate(path []string, data []byte) error {
	t.Lock()
	defer t.Unlock()
	uri := strings.Join(path, "/")

	switch uri {
	case DeviceEndpoint:
		// Got list of devices
		return nil
	case GroupEndpoint:
		// Got list of groups
		return nil
	case SceneEndpoint:
		// Got list of scenes
		return nil
	case GatewayEndpoint:
		return update(data, t.Gateway)
	case NotificationEndpoint:
		err := json.Unmarshal(data, &t.Notifications)
		if err != nil {
			return err
		}
		return nil
	}

	if len(path) == 1 {
		return fmt.Errorf("Got data at unknown endpoint %s: %s", uri, string(data))
	}

	id, err := strconv.Atoi(path[1])
	if err != nil {
		return fmt.Errorf("Expected int as second param, got %s", path[1])
	}

	var ok bool
	switch path[0] {
	case DeviceEndpoint:
		var d *Accessory
		if d, ok = t.Devices[id]; !ok {
			d = &Accessory{}
			d.InstanceID = id
			d.tree = t
			t.Devices[id] = d
			defer func() {
				for _, v := range t.callback {
					v.OnNewAccessory(d)
				}
			}()
		}
		return update(data, d)
	case GroupEndpoint:
		var d *Group
		var isNew bool
		if d, ok = t.Groups[id]; !ok {
			d = &Group{Scenes: map[int]*Scene{}}
			isNew = true
			t.Groups[id] = d
		}
		if d.tree == nil {
			isNew = true
		}
		if isNew {
			d.tree = t
			d.InstanceID = id
			defer func() {
				for _, v := range t.callback {
					v.OnNewGroup(d)
				}
			}()
		}
		return update(data, d)
	case SceneEndpoint:
		if len(path) == 3 {
			sceneId, err := strconv.Atoi(path[2])
			if err != nil {
				return fmt.Errorf("Expected int as third param, got %s", path[2])
			}
			var grp *Group
			if grp, ok = t.Groups[id]; !ok {
				grp = &Group{}
				t.Groups[id] = grp
			}
			if grp.Scenes == nil {
				grp.Scenes = map[int]*Scene{}
			}

			var scn *Scene
			var ok bool
			if scn, ok = grp.Scenes[sceneId]; !ok {
				scn = &Scene{}
				scn.group = grp
				scn.InstanceID = sceneId
				scn.tree = t
				grp.Scenes[sceneId] = scn
				defer func() {
					for _, v := range t.callback {
						v.OnNewScene(grp, scn)
					}
				}()
			}

			return update(data, scn)
		}
		return nil
	default:
		return fmt.Errorf("Got data at unknown endpoint %s: %s", uri, string(data))
	}
}
