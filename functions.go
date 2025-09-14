// functions.go
package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// разбираем на интерфейс, адрес и порт (1234)
func parseUdpAddr(udpAddr string) (*net.Interface, string, int, error) {
	re := regexp.MustCompile(`^udp://([^@]*)@([0-9.]+)(?::(\d+))?$`)
	matches := re.FindStringSubmatch(udpAddr)
	if len(matches) != 4 || !isValidIPv4(matches[2]) {
		return nil, "", 0, errors.New("Invalid address format: " + udpAddr)
	}
	ifi, err := net.InterfaceByName(matches[1])
	if matches[1] == "" || err != nil {
		ifi = nil
	}
	port, err := strconv.Atoi(matches[3])
	if err != nil || (port < 100 || port > 65535) {
		port = 1234
	}
	return ifi, matches[2], port, nil
}

// проверка на валидность ipv4 адреса..
func isValidIPv4(ip string) bool {
	re := regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	return re.MatchString(ip)
}

func parsePositionalArgs(args []string) (addr, token string, err error) {
	for _, a := range args {
		if strings.HasPrefix(a, "udp://") {
			if addr != "" {
				return "", "", fmt.Errorf("duplicated address argument: %q", a)
			}
			addr = a
		} else {
			if token != "" {
				return "", "", fmt.Errorf("duplicated token argument: %q", a)
			}
			token = a
		}
	}
	// оба параметра опциональны, но предупредим, если нет ни одного
	if token == "" {
		return "", "", fmt.Errorf("no token provided")
	}
	return addr, token, nil
}

func ifiName(ifi *net.Interface) string {
	if ifi == nil {
		return "(auto)"
	}
	return ifi.Name
}

// FetchMaybeGunzip скачивает URL и возвращает []byte.
// Если тело в gzip, распаковывает на лету. Есть таймаут и мягкий лимит на размер.
func FetchMaybeGunzip(ctx context.Context, url string, timeout time.Duration, softLimit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Нормальный UA помогает некоторым сервером
	req.Header.Set("User-Agent", "epgplayer/1.0 (+https://epg.by)")

	// В Go стандартный транспорт автоматически распаковывает *только* если сервер
	// отвечает Content-Encoding: gzip. Но если файл "лежит" gz-нутым без этого заголовка,
	// распаковки не будет — поэтому ниже делаем собственную проверку сигнатуры.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	// Оборачиваем в буфер, чтобы Peek-нуть первые байты
	br := bufio.NewReader(resp.Body)

	// Попробуем определить gzip по сигнатуре 1F 8B
	sig, _ := br.Peek(2) // Peek не потребляет байты
	isGzip := len(sig) >= 2 && sig[0] == 0x1f && sig[1] == 0x8b

	var r io.Reader = br
	if isGzip {
		gzr, err := gzip.NewReader(br)
		if err != nil {
			return nil, fmt.Errorf("gzip.NewReader: %w", err)
		}
		defer gzr.Close()
		r = gzr
	}

	// Если нужно ограничить объём, чтобы не съесть всю память:
	if softLimit > 0 {
		r = io.LimitReader(r, softLimit+1)
	}

	buf := bytes.NewBuffer(make([]byte, 0, 1<<20)) // стартовый кап 1 МБ
	n, err := buf.ReadFrom(r)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if softLimit > 0 && n > softLimit {
		return nil, fmt.Errorf("response exceeds soft limit (%d bytes > %d)", n, softLimit)
	}

	return buf.Bytes(), nil
}

func sleepUntilNext5Min() {
	now := time.Now()
	// ближайшая отметка следующей пятиминутки
	next := now.Truncate(5 * time.Minute).Add(5 * time.Minute)
	// выровняем по секундам: ждём до точно "…:..:00"
	next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), next.Minute(), 0, 0, next.Location())
	time.Sleep(time.Until(next))
}
