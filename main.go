package main

import (
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)

	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)

	if !ok {
		http.Error(w, "Hijacking is not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()

	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func handleHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(req)

	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, val := range values {
			dst.Add(key, val)
		}
	}
}

func main() {
	var pemPath string
	var keyPath string
	var proto string

	flag.StringVar(&pemPath, "pem", "server.pem", "path to pem file")
	flag.StringVar(&keyPath, "key", "server.key", "path to key file")
	flag.StringVar(&proto, "proto", "https", "Proxy protocol (http or https)")

	flag.Parse()

	if proto != "http" && proto != "https" {
		log.Fatal("Protocol must be either http or https")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			handleTunneling(w, r)
		} else {
			handleHTTP(w, r)
		}
	})

	tlsNP := make(map[string]func(*http.Server, *tls.Conn, http.Handler))

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		TLSNextProto: tlsNP,
	}

	if proto == "https" {
		log.Fatal(server.ListenAndServeTLS(pemPath, keyPath))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}
