package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
)

var (
	kubeletAddr = flag.String("kubelet_addr", "localhost:10250", "Address of local kubelet")
)

func main() {
	flag.Parse()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	for ; ; time.Sleep(time.Second) {
		ping(client)
	}
}

func ping(client *http.Client) {
	podsURL := fmt.Sprintf("%s/pods", *kubeletAddr)
	resp, err := client.Get(podsURL)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	var pl corev1.PodList
	if err := json.NewDecoder(resp.Body).Decode(&pl); err != nil {
		log.Fatal(err)
	}

	for _, p := range pl.Items {
		if p.Namespace == "system" && p.Name == "watcher" {
			continue
		}
		b, _ := json.MarshalIndent(p, "", "  ")
		log.Println(string(b))
	}
}
