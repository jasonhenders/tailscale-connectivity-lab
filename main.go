package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"tailscale.com/tsnet"
)

type requestInfo struct {
	Time       string              `json:"time"`
	Method     string              `json:"method"`
	Host       string              `json:"host"`
	URL        string              `json:"url"`
	RemoteAddr string              `json:"remote_addr"`
	UserAgent  string              `json:"user_agent"`
	Headers    map[string][]string `json:"headers"`
	TLS        bool                `json:"tls"`
}

func main() {
	mode := getenv("LAB_MODE", "tailnet") // tailnet or funnel
	hostname := getenv("TSNET_HOSTNAME", "funnel-lab-jason-1")
	authKey := os.Getenv("TS_AUTHKEY")

	s := &tsnet.Server{
		Dir:      "./tsnet-state",
		Hostname: hostname,
		AuthKey:  authKey,
	}
	defer s.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	mux.HandleFunc("/debug/request", func(w http.ResponseWriter, r *http.Request) {
		info := requestInfo{
			Time:       time.Now().Format(time.RFC3339Nano),
			Method:     r.Method,
			Host:       r.Host,
			URL:        r.URL.String(),
			RemoteAddr: r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Headers:    r.Header,
			TLS:        r.TLS != nil,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)

		log.Printf("%s %s host=%s remote=%s tls=%v",
			r.Method, r.URL.String(), r.Host, r.RemoteAddr, r.TLS != nil)
	})

	mux.HandleFunc("/debug/whoami", func(w http.ResponseWriter, r *http.Request) {
		lc, err := s.LocalClient()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		who, err := lc.WhoIs(context.Background(), r.RemoteAddr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(who)
	})

	var ln net.Listener
	var err error

	switch mode {
	case "tailnet":
		log.Printf("starting private tailnet listener on http://%s:8080", hostname)
		ln, err = s.Listen("tcp", ":8080")
	case "funnel":
		log.Printf("starting public Funnel listener on https://%s.appaloosa-ghost.ts.net", hostname)
		ln, err = s.ListenFunnel("tcp", ":443")
	default:
		log.Fatalf("unknown LAB_MODE %q; use tailnet or funnel", mode)
	}

	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	log.Fatal(http.Serve(ln, mux))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
