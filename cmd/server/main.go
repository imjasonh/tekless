package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/imjasonh/tekless/pkg"
	"github.com/imjasonh/tekless/pkg/storage"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/pod"
)

var (
	tok = flag.String("tok", "", "OAuth token") // TODO use ADC

	project     = flag.String("project", "", "Project to own the VM")
	zone        = flag.String("zone", "us-east4-a", "Zone to create the VM")
	machineType = flag.String("machine_type", "n1-standard-1", "Machine type")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	dsClient, err := storage.NewTaskRunStorage(ctx, *project)
	if err != nil {
		log.Fatalf("storage.NewTaskRunStorage: %v", err)
	}

	srv := &server{
		storage: dsClient,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("serving TaskRun API on port", port)
	// TODO: support namespaces as projects
	http.Handle("/apis/tekton.dev/v1beta1/default/taskruns", srv)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Support images from v0.14.2
var images = pipeline.Images{
	EntrypointImage: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/entrypoint:v0.14.2@sha256:3db5b1622b939b11603b49916cdfb5718e25add7a9c286a2832afb16f57f552f",
	CredsImage:      "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/creds-init:v0.14.2@sha256:64a511c0e68f611f630ccbe3744c97432f69c300940f4a32e748457581394dd6",
	NopImage:        "tianon/true@sha256:009cce421096698832595ce039aa13fa44327d96beedb84282a69d3dbcf5a81b",
	ShellImage:      "gcr.io/distroless/base@sha256:f79e093f9ba639c957ee857b1ad57ae5046c328998bf8f72b30081db4d8edbe4",
}

type server struct {
	storage *storage.TaskRunStorage
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/apis/tekton.dev/v1beta1/default/taskruns") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
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

	if err := pkg.RunPod(ctx, *p, *tok, *project, *zone, *machineType); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: persist in Datastore
	// TODO: respond with TaskRun JSON
}
