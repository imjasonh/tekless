package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/pod"
	"golang.org/x/oauth2"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	corev1 "k8s.io/api/core/v1"
)

var (
	file = flag.String("f", "", "Name of Pod spec to send to VM")

	tok = flag.String("tok", "", "OAuth token") // TODO use ADC

	project     = flag.String("project", "", "Project to own the VM")
	zone        = flag.String("zone", "us-east4-a", "Zone to create the VM")
	machineType = flag.String("machine_type", "n1-standard-1", "Machine type")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	if *file != "" {
		log.Println("running pod spec:", *file)

		b, err := ioutil.ReadFile(*file)
		if err != nil {
			log.Fatal(err)
		}

		var pod corev1.Pod
		if err := yaml.Unmarshal(b, &pod); err != nil {
			log.Fatal(err)
		}

		if err := runPod(ctx, pod); err != nil {
			log.Fatal(err)
		}
		return
	}

	// TODO: serve TaskRun creation API
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("serving TaskRun API on port", port)
	// TODO: support namespaces as projects
	http.HandleFunc("/apis/tekton.dev/v1beta1/default/taskruns", handle)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Support images from v0.14.2
var images = pipeline.Images{
	EntrypointImage: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/entrypoint:v0.14.2@sha256:3db5b1622b939b11603b49916cdfb5718e25add7a9c286a2832afb16f57f552f",
	CredsImage:      "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/creds-init:v0.14.2@sha256:64a511c0e68f611f630ccbe3744c97432f69c300940f4a32e748457581394dd6",
	NopImage:        "tianon/true@sha256:009cce421096698832595ce039aa13fa44327d96beedb84282a69d3dbcf5a81b",
	ShellImage:      "gcr.io/distroless/base@sha256:f79e093f9ba639c957ee857b1ad57ae5046c328998bf8f72b30081db4d8edbe4",
}

func handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "must be POST", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	defer r.Body.Close()
	var tr *v1beta1.TaskRun
	if err := json.NewDecoder(r.Body).Decode(tr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate TaskRun request (must inline taskSpec)
	// TODO: support referencing Tasks
	if err := tr.Validate(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if tr.Spec.TaskSpec == nil {
		http.Error(w, "must inline taskSpec", http.StatusBadRequest)
		return
	}

	p, err := pod.MakePod(ctx, images, tr, *tr.Spec.TaskSpec, nil, nil, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := runPod(ctx, *p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: persist in Datastore
	// TODO: respond with TaskRun JSON
}

func runPod(ctx context.Context, pod corev1.Pod) error {
	svc, err := compute.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *tok})))
	if err != nil {
		return err
	}

	b, err := json.Marshal(pod)
	if err != nil {
		return err
	}
	podstr := string(b)

	region := (*zone)[:strings.LastIndex(*zone, "-")]

	name := "instance-" + uuid.New().String()[:4]
	log.Printf("creating %q...", name)
	op, err := svc.Instances.Insert(*project, *zone, &compute.Instance{
		Name:        name,
		Zone:        *zone,
		MachineType: fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", *project, *zone, *machineType),
		Disks: []*compute.AttachedDisk{{
			InitializeParams: &compute.AttachedDiskInitializeParams{
				SourceImage: "projects/cos-cloud/global/images/family/cos-stable",
			},
			Boot: true,
		}},
		NetworkInterfaces: []*compute.NetworkInterface{{
			Subnetwork: fmt.Sprintf("projects/%s/regions/%s/subnetworks/default", *project, region),
			AccessConfigs: []*compute.AccessConfig{{
				Name:        "External NAT",
				Type:        "ONE_TO_ONE_NAT",
				NetworkTier: "PREMIUM",
			}},
		}},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{{
				Key:   "user-data",
				Value: &cloudConfig,
			}, {
				Key:   "pod",
				Value: &podstr,
			}, {
				Key:   "ca-cert",
				Value: &caCert,
			}, {
				Key:   "ca-cert-key",
				Value: &caCertKey,
			}, {
				Key:   "cos-metrics-enabled",
				Value: &trueString,
			}},
		},
		Tags: &compute.Tags{Items: []string{"https-server"}},
		// TODO: enable secure boot
	}).Do()
	if err != nil {
		return err
	}

	start := time.Now()
	for ; ; time.Sleep(time.Second) {
		op, err = svc.ZoneOperations.Get(*project, *zone, op.Name).Do()
		if err != nil {
			return err
		}
		log.Printf("operation is %q...", op.Status)
		if op.Status == "DONE" {
			break
		}
	}
	log.Println("startup took", time.Since(start))
	return nil
}
