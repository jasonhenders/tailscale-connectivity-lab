package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

var proxyV2Signature = []byte{
	0x0d, 0x0a, 0x0d, 0x0a,
	0x00, 0x0d, 0x0a, 0x51,
	0x55, 0x49, 0x54, 0x0a,
}

type proxyListener struct {
	net.Listener
}

type proxyConn struct {
	net.Conn
	reader     *bufio.Reader
	remoteAddr net.Addr
	localAddr  net.Addr
}

func (c *proxyConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *proxyConn) RemoteAddr() net.Addr {
	if c.remoteAddr != nil {
		return c.remoteAddr
	}
	return c.Conn.RemoteAddr()
}

func (c *proxyConn) LocalAddr() net.Addr {
	if c.localAddr != nil {
		return c.localAddr
	}
	return c.Conn.LocalAddr()
}

func (l *proxyListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	br := bufio.NewReader(c)

	remoteAddr, localAddr, err := parseProxyV2(br)
	if err != nil {
		_ = c.Close()
		return nil, err
	}

	return &proxyConn{
		Conn:       c,
		reader:     br,
		remoteAddr: remoteAddr,
		localAddr:  localAddr,
	}, nil
}

func parseProxyV2(br *bufio.Reader) (net.Addr, net.Addr, error) {
	hdr, err := br.Peek(16)
	if err != nil {
		return nil, nil, err
	}

	// If this is not PROXY protocol v2, leave the stream untouched.
	// This allows local direct HTTP testing without a PROXY header.
	if !bytes.HasPrefix(hdr, proxyV2Signature) {
		return nil, nil, nil
	}

	_, _ = br.Discard(16)

	verCmd := hdr[12]
	famProto := hdr[13]
	length := int(binary.BigEndian.Uint16(hdr[14:16]))

	if verCmd>>4 != 0x2 {
		return nil, nil, fmt.Errorf("not PROXY protocol v2")
	}

	cmd := verCmd & 0x0f
	if cmd != 0x01 {
		// LOCAL command. Keep original socket addresses.
		if length > 0 {
			_, _ = io.CopyN(io.Discard, br, int64(length))
		}
		return nil, nil, nil
	}

	addr := make([]byte, length)
	if _, err := io.ReadFull(br, addr); err != nil {
		return nil, nil, err
	}

	switch famProto {
	case 0x11: // TCP over IPv4
		if len(addr) < 12 {
			return nil, nil, fmt.Errorf("short IPv4 PROXY header")
		}

		srcIP := net.IP(addr[0:4])
		dstIP := net.IP(addr[4:8])
		srcPort := int(binary.BigEndian.Uint16(addr[8:10]))
		dstPort := int(binary.BigEndian.Uint16(addr[10:12]))

		return &net.TCPAddr{IP: srcIP, Port: srcPort},
			&net.TCPAddr{IP: dstIP, Port: dstPort},
			nil

	case 0x21: // TCP over IPv6
		if len(addr) < 36 {
			return nil, nil, fmt.Errorf("short IPv6 PROXY header")
		}

		srcIP := net.IP(addr[0:16])
		dstIP := net.IP(addr[16:32])
		srcPort := int(binary.BigEndian.Uint16(addr[32:34]))
		dstPort := int(binary.BigEndian.Uint16(addr[34:36]))

		return &net.TCPAddr{IP: srcIP, Port: srcPort},
			&net.TCPAddr{IP: dstIP, Port: dstPort},
			nil

	default:
		return nil, nil, fmt.Errorf("unsupported PROXY v2 family/proto: 0x%x", famProto)
	}
}

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
	addr := ":9899"

	baseLn, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	ln := &proxyListener{Listener: baseLn}

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

	log.Printf("PROXY protocol test backend listening on %s", addr)
	log.Fatal(http.Serve(ln, mux))
}
