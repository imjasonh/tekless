package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
)

var (
	kubeletAddr = flag.String("kubelet_addr", "https://localhost:10250", "Address of local kubelet")

	apiHost = flag.String("api_host", "", "Cloudu Run HTTPS host to send updates")
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

	prev := corev1.PodStatus{}
	for ; ; time.Sleep(time.Second) {
		podsURL := fmt.Sprintf("%s/pods", *kubeletAddr)
		resp, err := client.Get(podsURL)
		if err != nil {
			log.Fatal(err)
		}

		var pl corev1.PodList
		if err := json.NewDecoder(resp.Body).Decode(&pl); err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()

		for _, p := range pl.Items {
			if p.Namespace == "system" {
				continue
			}

			if d := cmp.Diff(p.Status, prev); d != "" {
				log.Println("Pod update (-was,+now):", d)
				prev = p.Status
			}
		}
	}
}
