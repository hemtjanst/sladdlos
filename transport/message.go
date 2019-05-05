package transport

import (
	"encoding/json"
	"fmt"
	"github.com/satori/go.uuid"
	"log"
	"time"
)

type tradfriRaw struct {
	Method     string          `json:"method"`
	URL        string          `json:"url"`
	ID         string          `json:"id,omitempty"`
	ReplyTopic string          `json:"replyTopic,omitempty"`
	Payload    json.RawMessage `json:"payload"`
}

type tradfriReply struct {
	ID      string          `json:"id"`
	Code    string          `json:"code"`
	Format  int             `json:"format"`
	Payload json.RawMessage `json:"payload"`
}

func (t *Transport) onReply(msg []byte) {
	resp := &tradfriReply{}
	err := json.Unmarshal(msg, resp)
	smsg := string(msg)
	if err != nil {
		log.Printf("Error (%s) in Trådfri reply: %v", err, smsg)
		return
	}
	if resp.ID == "" {
		log.Printf("Trådfri reply: got empty ID -  %s", smsg)
		return
	}

	var ch chan *tradfriReply
	var ok bool
	t.RLock()
	defer t.RUnlock()
	if ch, ok = t.waiting[resp.ID]; !ok {
		log.Printf("Trådfri reply: got reply for unknown ID - %s", smsg)
		return
	}
	go func() {
		// Make sure we don't end up blocking forever by reading
		// off the channel when the other end has given up.
		t := time.AfterFunc(2*time.Second, func() {
			<-ch
		})
		ch <- resp
		t.Stop()
	}()
}

func (t *Transport) makeReq(method, uri string, payload []byte) ([]byte, error) {
	ch := make(chan *tradfriReply)
	id := uuid.NewV4().String()
	t.Lock()
	t.waiting[id] = ch
	t.Unlock()
	defer func() {
		t.Lock()
		defer t.Unlock()
		delete(t.waiting, id)
	}()

	req := &tradfriRaw{
		Method:     method,
		URL:        uri,
		ReplyTopic: "tradfri-reply/" + t.id,
		ID:         id,
	}
	if payload != nil {
		req.Payload = json.RawMessage(payload)
	}
	js, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	t.client.Publish("tradfri-cmd", js, false)

	select {
	case r := <-ch:
		return r.Payload, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout after 10 seconds")
	}
}

func (t *Transport) Get(uri string) ([]byte, error) {
	return t.makeReq("get", uri, nil)
}

func (t *Transport) Put(uri string, data []byte) error {
	_, err := t.makeReq("put", uri, data)
	return err
}

func (t *Transport) Delete(uri string) error {
	_, err := t.makeReq("delete", uri, nil)
	return err
}
