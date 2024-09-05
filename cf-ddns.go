package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type DNSRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
}

type DNSResponse struct {
	Status int `json:"Status"`
	Answer []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
		TTL  int    `json:"TTL"`
		Data string `json:"data"`
	} `json:"Answer"`
}

func main() {

	var apiToken, zoneID, dnsRecordID, dnsRecordName string

	// Define command-line flags
	flag.StringVar(&apiToken, "api-token", os.Getenv("CLOUDFLARE_API_TOKEN"), "Cloudflare API Token")
	flag.StringVar(&zoneID, "zone-id", os.Getenv("CLOUDFLARE_ZONE_ID"), "Cloudflare Zone ID")
	flag.StringVar(&dnsRecordID, "dns-record-id", os.Getenv("CLOUDFLARE_DNS_RECORD_ID"), "Cloudflare DNS Record ID")
	flag.StringVar(&dnsRecordName, "dns-record-name", os.Getenv("CLOUDFLARE_DNS_RECORD_NAME"), "Cloudflare DNS Record Name")

	flag.Parse()

	if apiToken == "" || zoneID == "" || dnsRecordID == "" || dnsRecordName == "" {
		fmt.Println("ERROR: Please set all the required environment vars or provide them as command line args.")
		fmt.Println("Usage: cf-ddns")
		fmt.Println("\t-api-token <ZONE_ID>\t\tor set environment var CLOUDFLARE_ZONE_ID")
		fmt.Println("\t-api-token <API_TOKEN>\t\tor set environment var CLOUDFLARE_API_TOKEN")
		fmt.Println("\t-api-token <DNS_RECORD_ID>\tor set environment var CLOUDFLARE_DNS_RECORD_ID")
		fmt.Println("\t-api-token <DNS_RECORD_NAME>\tor set environment var CLOUDFLARE_DNS_RECORD_NAME")
		return
	}

	ip, err := getPublicIPv4()
	if err != nil {
		slog.Error("Error getting public IP:", err)
		return
	}

	slog.Info("", "Public IP", ip)

	currIP, err := getCurrentIPv4(dnsRecordName)
	if err != nil {
		slog.Error("Error getting current IP:", err)
		return
	}

	slog.Info("", "Current IP", currIP)

	if ip == currIP {
		slog.Info("Nothing to update, exiting.")
		return
	}

	err = updateDNSRecord(apiToken, zoneID, dnsRecordID, dnsRecordName, ip)
	if err != nil {
		slog.Error("Error updating DNS record:", err)
		return
	}

	slog.Info("DNS record updated successfully.")

	return
}

func getPublicIPv4() (string, error) {
	url := "https://cloudflare.com/cdn-cgi/trace"

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("tcp4", addr)
			},
		},
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	output := string(body)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "ip=") {
			return strings.TrimPrefix(line, "ip="), nil
		}
	}

	return "", fmt.Errorf("IP address not found")
}

func updateDNSRecord(apiToken, zoneID, dnsRecordID, dnsRecordName, ip string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, dnsRecordID)

	record := DNSRecord{
		Content: ip,
		Name:    dnsRecordName,
	}

	jsonData, err := json.Marshal(record)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update DNS record: %s", body)
	}

	return nil
}

func getCurrentIPv4(name string) (string, error) {
	url := fmt.Sprintf("https://cloudflare-dns.com/dns-query?name=%s&type=A", name)

	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/dns-json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: received non-200 response status:", resp.Status)
		return "", errors.New("Non-200 response status")
	}

	// Parse the JSON response
	var dnsResponse DNSResponse
	if err := json.NewDecoder(resp.Body).Decode(&dnsResponse); err != nil {
		fmt.Println("Error decoding response:", err)
		return "", err
	}

	// Check the status of the response
	if dnsResponse.Status != 0 {
		fmt.Printf("Error: DNS query failed with status %d\n", dnsResponse.Status)
		return "", errors.New("DNS query failed")
	}

	// return the first record
	return dnsResponse.Answer[0].Data, nil
}
