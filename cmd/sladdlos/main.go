package main

import (
	"context"
	"flag"
	"github.com/satori/go.uuid"
	"hemtjan.st/sladdlos"
	"hemtjan.st/sladdlos/tradfri"
	"hemtjan.st/sladdlos/transport"
	"lib.hemtjan.st/client"
	"lib.hemtjan.st/server"
	"lib.hemtjan.st/transport/mqtt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	cleanUpHemtjanst = flag.Bool("hemtjanst.cleanup", false, "Clean up Hemtjänst MQTT Topics")
	cleanUpTradfri   = flag.Bool("tradfri.cleanup", false, "Clean up Trådfri MQTT Topics")
	skipGroup        = flag.Bool("skip-group", false, "Skip announcing Trådfri groups as lights")
	skipBulb         = flag.Bool("skip-bulb", false, "Skip announcing Trådfri bulbs individually")
)

func main() {

	mCfg := mqtt.MustFlags(flag.String, flag.Bool)

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mq, err := mqtt.New(ctx, mCfg())
	if err != nil {
		log.Fatal(err)
	}

	if *cleanUpHemtjanst || *cleanUpTradfri {
		clean(mq, ctx, cancel)
		return
	}

	if *skipGroup && *skipBulb {
		log.Print("-skip-group and -skip-bulb are mutually exclusive, pick one")
		return
	}

	go func() {
		quit := make(chan os.Signal)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		<-quit
		cancel()
	}()

	id := uuid.NewV4().String()

	tr := transport.NewTransport(mq, id)
	tree := tradfri.NewTree(tr)
	tr.SetTree(tree)

	ht := sladdlos.NewHemtjanstClient(tree, mq, id)

	ht.SkipGroup = *skipGroup
	if ht.SkipGroup {
		log.Print("Skipping groups")
	}
	ht.SkipBulb = *skipBulb
	if ht.SkipBulb {
		log.Print("Skipping bulbs")
	}

	ht.Start(ctx)

	<-ctx.Done()
}

func clean(tr mqtt.MQTT, ctx context.Context, cancel func()) {
	time.AfterFunc(10*time.Second, cancel)

	if *cleanUpTradfri {
		rawch := tr.SubscribeRaw("tradfri-raw/#")

		for {
			m, open := <-rawch
			if !open {
				return
			}
			if m.IsRetain {
				log.Printf("Deleting contents of topic %s", m.TopicName)
				tr.Publish(m.TopicName, []byte{}, true)
			}

		}
	}

	if *cleanUpHemtjanst {

		srv := server.New(tr)
		ch := make(chan server.Update, 5)
		srv.SetUpdateChannel(ch)
		go func() {
			for {
				ev, open := <-ch
				if !open {
					return
				}
				if ev.Type != server.AddedDevice {
					continue
				}
				sp := strings.Split(ev.Device.Id(), "/")
				if len(sp) == 2 && (strings.Index(sp[1], "grp-") == 0 || strings.Index(sp[1], "bulb-") == 0) {
					log.Printf("Deleting device %s", ev.Device.Id())
					_ = client.DeleteDevice(ev.Device.Info(), tr)
				}

			}
		}()
		_ = srv.Start(ctx)
	}

}
