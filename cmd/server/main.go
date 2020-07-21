package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"github.com/imjasonh/tekless/pkg"
	"github.com/imjasonh/tekless/pkg/storage"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	watcherImage = flag.String("watcher_image", "", "Image of watcher container")

	tok = flag.String("tok", "", "OAuth token") // TODO use ADC

	project     = flag.String("project", "", "Project to own the VM")
	zone        = flag.String("zone", "us-east4-a", "Zone to create the VM")
	machineType = flag.String("machine_type", "n1-standard-1", "Machine type")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	if *project == "" {
		p, err := metadata.ProjectID()
		if err != nil {
			log.Fatal("couldn't determine project ID from metadata")
		}
		*project = p
	}

	var ts oauth2.TokenSource
	if *tok != "" {
		ts = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *tok})
	} else {
		ts = google.ComputeTokenSource("", "https://www.googleapis.com/auth/cloud-platform")
	}

	dsClient, err := storage.NewTaskRunStorage(ctx, *project)
	if err != nil {
		log.Fatalf("storage.NewTaskRunStorage: %v", err)
	}

	srv := &server{
		ts:      ts,
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

type server struct {
	ts      oauth2.TokenSource
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
	var tr v1beta1.TaskRun
	if err := json.NewDecoder(r.Body).Decode(&tr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := pkg.RunTaskRun(ctx, tr, s.ts, *watcherImage, *project, *zone, *machineType); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: persist in Datastore
	// TODO: respond with TaskRun JSON
}
