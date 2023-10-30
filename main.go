package main

import (
	"flag"
	"net/http"

	"go.uber.org/zap"
)

func main() {
	log, _ := zap.NewDevelopment()
	addr := flag.String("addr", ":9981", "comchat server listen address")
	s, err := NewServer(log, "")
	if err != nil {
		log.Fatal("NewServer failed: %v", zap.Error(err))
	}
	log.Info("start run", zap.String("address", *addr))
	_ = http.ListenAndServe(*addr, s)
}
