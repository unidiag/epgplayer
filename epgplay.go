package main

import (
	"errors"
	"io"
	"log"
	"net"
	"time"
)

func epgPlay(addrArg string, f io.ReadSeeker) {

	// буфер кратный 188 — собираем по 7 пакетов в одну датаграмму
	buf := make([]byte, udpPayloadTS)

	ifi, ipStr, port, err := parseUdpAddr(addrArg)
	if err != nil {
		log.Fatalf("bad udp address %q: %v", addrArg, err)
	}
	ip := net.ParseIP(ipStr)

	conn, err := openSocket4(ifi, ip, port)
	if err != nil {
		log.Fatalf("open UDP socket failed: %v", err)
	}
	defer conn.Close()

	dest := &net.UDPAddr{IP: ip, Port: port}

	stopPlayer = false

	for {

		if stopPlayer {
			return
		}

		n, rerr := io.ReadFull(f, buf)
		if rerr != nil {
			if errors.Is(rerr, io.ErrUnexpectedEOF) {
				// округлим вниз до кратности 188
				n -= n % pktSize
			} else if errors.Is(rerr, io.EOF) {
				// зацикливаем
				if _, err := f.Seek(0, io.SeekStart); err != nil {
					log.Fatalf("seek failed: %v", err)
				}
				continue
			} else {
				log.Fatalf("read failed: %v", rerr)
			}
		}
		if n <= 0 {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				log.Fatalf("seek failed: %v", err)
			}
			continue
		}
		n -= n % pktSize
		if n <= 0 {
			continue
		}

		if _, err := conn.WriteTo(buf[:n], dest); err != nil {
			log.Printf("udp write error: %v", err)
		}

		// лёгкий темп, чтобы не залить сеть (упростим без знания битрейта)
		time.Sleep(pause)

	}
}
