package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"

	"github.com/ghodss/yaml"
	"github.com/imjasonh/tekless/pkg"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
)

var (
	watcherImage = flag.String("watcher_image", "", "Image of watcher container")
	file         = flag.String("f", "", "Name of Pod spec to send to VM")

	tok = flag.String("tok", "", "OAuth token") // TODO use ADC

	project     = flag.String("project", "", "Project to own the VM")
	zone        = flag.String("zone", "us-east4-a", "Zone to create the VM")
	machineType = flag.String("machine_type", "n1-standard-1", "Machine type")
)

func main() {
	flag.Parse()
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *tok})

	log.Println("running pod spec:", *file)

	b, err := ioutil.ReadFile(*file)
	if err != nil {
		log.Fatal(err)
	}

	var pod corev1.Pod
	if err := yaml.Unmarshal(b, &pod); err != nil {
		log.Fatal(err)
	}

	if err := pkg.RunPod(ctx, pod, ts, *watcherImage, *project, *zone, *machineType); err != nil {
		log.Fatal(err)
	}
}
