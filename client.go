package main

import (
	"context"
	"encoding/json"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"io"
	"log"
	"time"
)

func Client() error {
	ctx := context.Background()
	conn, _, _, err := ws.DefaultDialer.Dial(ctx, "ws://localhost:8080")
	if err != nil {
		return err
	}

	go func() {
		for {
			msg, _, err := wsutil.ReadServerData(conn)
			if err != nil {
				log.Print(err)
			}

			log.Print(string(msg))
		}
	}()

	ew := errWriter{Writer: conn}

	wsutil.WriteClientMessage(&ew, ws.OpText, encode(Flour{100}))
	wsutil.WriteClientMessage(&ew, ws.OpText, encode(Flour{450}))
	wsutil.WriteClientMessage(&ew, ws.OpText, encode(Eggs{4}))
	wsutil.WriteClientMessage(&ew, ws.OpText, encode(Milk{1.5}))
	wsutil.WriteClientMessage(&ew, ws.OpText, encode(Flour{450}))
	if ew.err == nil {
		time.Sleep(time.Second * 5)
	}
	wsutil.WriteClientMessage(&ew, ws.OpText, encode(Eggs{2}))
	wsutil.WriteClientMessage(&ew, ws.OpText, encode(Flour{300}))
	wsutil.WriteClientMessage(&ew, ws.OpText, encode(Milk{1.5}))
	if ew.err == nil {
		time.Sleep(time.Second * 10)
	}

	return ew.err
}

func encode(i interface{}) []byte {
	if res, err := json.Marshal(i); err == nil {
		return res
	}
	return []byte("{}")
}

type errWriter struct {
	io.Writer
	err error
}

func (e *errWriter) Write(buf []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}
	var n int
	n, e.err = e.Writer.Write(buf)
	return n, nil
}
