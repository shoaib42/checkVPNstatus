package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

var dnsServer string

func main() {

	port := flag.Int("port", 8080, "Port number to bind to")
	dnsServerIP := flag.String("dns", "8.8.8.8", "DNS server IP address to use")
	flag.Parse()

	dnsServer = fmt.Sprintf("%s:53", *dnsServerIP)

	http.HandleFunc("/checkvpn", checkVPNHandler)

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("Server listening on port %d\n", *port)
	fmt.Printf("DNS server set to %s\n", dnsServer)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
		os.Exit(1)
	}
}

func checkVPNHandler(w http.ResponseWriter, r *http.Request) {

	domain := r.URL.Query().Get("domain")
	if domain == "" {
		http.Error(w, "Missing 'domain' parameter", http.StatusBadRequest)
		return
	}

	ip, err := resolveWithDNS(domain, dnsServer)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error resolving %s: %s", domain, err), http.StatusInternalServerError)
		return
	}

	publicIP, err := getPublicIP()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching public IP: %s", err), http.StatusInternalServerError)
		return
	}

	vpnStatus := "VPN is not working"
	if ip != publicIP {
		vpnStatus = "VPN is working"
	}

	response := map[string]string{"status": vpnStatus}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON response: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonResponse)
}

func resolveWithDNS(hostname, dnsServer string) (string, error) {

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialer := net.Dialer{}
			return dialer.DialContext(ctx, "udp", dnsServer)
		},
	}

	ips, err := resolver.LookupIP(context.Background(), "ip", hostname)
	if err != nil {
		return "", err
	}

	return ips[0].String(), nil
}

func getPublicIP() (string, error) {
	resp, err := http.Get("https://ifconfig.me/ip")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}
