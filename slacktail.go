package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

// 512kb read chunk
const readChunkSize = 524288

type Spinner struct {
	currIdx int
}

func (s *Spinner) Frame() string {
	glyphs := [4]string{"|", "/", "-", "\\"}
	tmp := glyphs[s.currIdx]
	if s.currIdx == len(glyphs)-1 {
		s.currIdx = 0
	} else {
		s.currIdx = s.currIdx + 1
	}
	return tmp
}

type SlackClient struct {
	webhook string
	channel string
}

func (sc *SlackClient) Post(msg string) int {
	m := map[string]string{"text": msg, "channel": sc.channel}
	doc, _ := json.Marshal(m)
	http.PostForm(sc.webhook, url.Values{"payload": {string(doc)}})
	return 1
}

func ingest(f *os.File, textBuffer chan string, delay float64) {
	// Seek to EOF
	offset, _ := f.Seek(0, 2)

	spinner := Spinner{currIdx: 0}
	totalBytesRead := 0
	for {
		b := make([]byte, readChunkSize)
		bytesRead, _ := f.ReadAt(b, offset)

		glyph := spinner.Frame()
		if bytesRead != 0 {
			textBuffer <- string(bytes.Trim(b, "\x00"))
			glyph = "R"
			totalBytesRead = totalBytesRead + bytesRead
		}

		fmt.Printf("\r[%s] %d bytes read", glyph, totalBytesRead)

		offset = offset + int64(bytesRead)
		time.Sleep(time.Duration(delay*1000) * time.Millisecond)
	}
}

func flush(client SlackClient, textBuffer chan string, delay float64) {
	for {
		msg := <-textBuffer
		client.Post(msg)
		time.Sleep(time.Duration(delay*1000) * time.Millisecond)
	}
}

func main() {
	webhook := flag.String("webhook", "", "Slack webhook url")
	channel := flag.String("channel", "", "Channel name with leading #")
	ingestDelay := flag.Float64("ingestDelay", 1, "Channel name")
	flushDelay := flag.Float64("flushDelay", 5, "Channel name")
	flag.Parse()

	filepath := flag.Arg(0)

	if *webhook == "" || *channel == "" || filepath == "" {
		fmt.Println("Usage: slacktail -channel <channel> -webhook <webhook> filepath")
		os.Exit(1)
	}

	f, e := os.Open(filepath)

	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}

	textBuffer := make(chan string)
	client := SlackClient{webhook: *webhook, channel: *channel}

	fmt.Printf("%s -> %s. (Press ctrl-c to stop)\n", filepath, *channel)
	go ingest(f, textBuffer, *ingestDelay)
	go flush(client, textBuffer, *flushDelay)

	fmt.Scanln()
}
