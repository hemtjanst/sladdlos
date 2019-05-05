package transport

import (
	"hemtjan.st/sladdlos/tradfri"
	"lib.hemtjan.st/transport/mqtt"
	"log"
	"strings"
	"sync"
)

type Transport struct {
	sync.RWMutex
	client  mqtt.MQTT
	id      string
	tree    *tradfri.Tree
	waiting map[string]chan *tradfriReply
}

func NewTransport(mq mqtt.MQTT, id string) *Transport {
	m := &Transport{
		client:  mq,
		id:      id,
		waiting: map[string]chan *tradfriReply{},
	}
	return m
}

func (t *Transport) SetTree(tree *tradfri.Tree) {
	t.Lock()
	defer t.Unlock()
	if tree == nil {
		t.tree = nil
		if t.client != nil {
			t.client.Unsubscribe("tradfri-raw/#")
		}
		return
	}
	t.tree = tree
	if t.client != nil {
		t.subscribe()
	}
}

func (t *Transport) onMessage(msg *mqtt.Packet) {
	t.RLock()
	defer t.RUnlock()
	if t.tree == nil {
		return
	}
	topic := strings.Split(msg.TopicName, "/")
	if len(topic) < 2 || topic[0] != "tradfri-raw" {
		return
	}
	go func() {
		err := t.tree.Populate(topic[1:], msg.Payload)

		if err != nil {
			log.Print(err)
			log.Printf("^- While parsing from %s: %s", msg.TopicName, string(msg.Payload))
		}
	}()
}

func (t *Transport) subscribe() {
	msgCh := t.client.SubscribeRaw("tradfri-raw/#")
	rplCh := t.client.Subscribe("tradfri-reply/" + t.id)
	go func() {
		for {
			select {
			case msg, open := <-msgCh:
				if !open {
					return
				}
				t.onMessage(msg)
			case rpl, open := <-rplCh:
				if !open {
					return
				}
				t.onReply(rpl)

			}

		}
	}()
}
