// main.go
package main

import (
	"bytes"
	"context"
	"log"
	"os"
	"time"
)

const (
	defaultDst = "udp://lo@239.1.1.50:5500"
	// MPEG-TS
	pktSize      = 188
	udpPayloadTS = 7 * pktSize // 1316 байт в UDP-пакете
	pause        = 50 * time.Millisecond
)

var stopPlayer = false
var updateCnt = -1

func main() {

	addrArg, tokenArg, err := parsePositionalArgs(os.Args[1:])
	log.Println("************************************")
	log.Println("*** EPGPlayer for service EPG.BY ***")
	log.Println("************************************")
	if err != nil {
		log.Printf("Usage: %s <token> (udp://lo@239.1.1.50:5500)\n", os.Args[0])
		os.Exit(2)
	}

	if addrArg == "" {
		addrArg = defaultDst
	}

	log.Printf("[START] Streaming EIT to %s", addrArg)

	for {

		url := "http://epg.by/" + tokenArg + "/eit.ts"
		upd := "Repeat"
		// обновляем структуру на старте и каждые 3 часа
		// потом только репит (быстрее)
		if updateCnt >= 36 || updateCnt == -1 {
			url += "?r=1"
			upd = "Download"
			updateCnt = 0
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		data, err := FetchMaybeGunzip(ctx, url, 30*time.Second, 10<<20) // лимит 10 МБ
		if err != nil {
			log.Printf("fetch error: %v", err)
			time.Sleep(60 * time.Second)
			continue
		}

		if len(data) > 0 && data[0] == 0x47 {
			log.Printf("[INFO] %s EPG %d kbytes MPEG-TS\n", upd, len(data)/1024)
			time.Sleep(250 * time.Microsecond)
			r := bytes.NewReader(data)
			go epgPlay(addrArg, r)
			sleepUntilNext5Min()
			stopPlayer = true
		} else {
			text := string(data)
			if len(data) > 100 {
				text = string(data[:100])
			}
			log.Printf("[ERROR] Got %d bytes and its not MPEG-TS (retry 30 sec)\n%s...\n", len(data), text)
			time.Sleep(30 * time.Second)
		}
		updateCnt++
		cancel()
	}
}
