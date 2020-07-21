package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"

	"github.com/ghodss/yaml"
	"github.com/imjasonh/tekless/pkg"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
)

var (
	watcherImage = flag.String("watcher_image", "", "Image of watcher container")

	podfile = flag.String("pod", "", "Name of Pod YAML file to send to VM")
	trfile  = flag.String("taskrun", "", "Name of TaskRun YAML file to send to VM")

	tok = flag.String("tok", "", "OAuth token") // TODO use ADC

	project     = flag.String("project", "", "Project to own the VM")
	zone        = flag.String("zone", "us-east4-a", "Zone to create the VM")
	machineType = flag.String("machine_type", "n1-standard-1", "Machine type")
)

func main() {
	flag.Parse()

	if *podfile == "" && *trfile == "" {
		log.Fatal("must specify --pod or --taskrun")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *tok})

	if *podfile != "" {
		log.Println("running pod spec:", *podfile)

		b, err := ioutil.ReadFile(*podfile)
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
	} else if *trfile != "" {
		log.Println("running taskrun spec:", *trfile)

		b, err := ioutil.ReadFile(*trfile)
		if err != nil {
			log.Fatal(err)
		}

		var tr v1beta1.TaskRun
		if err := yaml.Unmarshal(b, &tr); err != nil {
			log.Fatal(err)
		}

		if err := pkg.RunTaskRun(ctx, tr, ts, *watcherImage, *project, *zone, *machineType); err != nil {
			log.Fatal(err)
		}
	}
}
