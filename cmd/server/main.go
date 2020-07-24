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
	"github.com/google/uuid"
	"github.com/imjasonh/tekless/pkg"
	"github.com/imjasonh/tekless/pkg/storage"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	watcherImage = flag.String("watcher_image", "", "Image of watcher container")
	// TODO: could look this up in the Cloud Run API...
	host = flag.String("host", "", "Cloud Run HTTPS host the watcher should POST to")

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
		log.Println("defaulting project to", *project)
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
	match := func(r *http.Request, method, path string) bool {
		return r.Method == method && strings.HasPrefix(r.URL.Path, path)
	}
	switch {
	case match(r, "POST", "/apis/tekton.dev/v1beta1/default/taskruns"):
		s.insert(w, r)
	case match(r, "PATCH", "/internal/taskruns/"):
		s.update(w, r)
	case match(r, "GET", "/apis/tekton.dev/v1betta1/default/taskruns/"):
		s.get(w, r)
	case match(r, "DELETE", "/apis/tekton.dev/v1beta1/default/taskruns/"):
		s.delete(w, r)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (s *server) insert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	defer r.Body.Close()
	var tr v1beta1.TaskRun
	if err := json.NewDecoder(r.Body).Decode(&tr); err != nil {
		log.Printf("json.Decode: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := tr.Validate(ctx); err != nil {
		log.Printf("invalid TaskRun: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If incoming request specifies generateName, generate a name.
	// Otherwise, check if it exists.
	if tr.Name == "" && tr.GenerateName != "" {
		tr.Name = tr.GenerateName + uuid.New().String()[:5]
		tr.GenerateName = ""
	} else if s.storage.Exists(ctx, tr.Name) {
		log.Printf("Insert for existing TaskRun %q", tr.Name)
		http.Error(w, "taskrun exists", http.StatusConflict)
		return
	}

	// Start a new VM to run the TaskRun.
	tr.CreationTimestamp = metav1.Now()
	log.Printf("starting new TaskRun %q", tr.Name)
	if err := pkg.RunTaskRun(ctx, tr, s.ts, *watcherImage, *project, *zone, *machineType); err != nil {
		log.Printf("RunTaskRun(%q): %v", tr.Name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("inserting %q", tr.Name)
	if err := s.storage.Insert(ctx, &tr); err != nil {
		log.Printf("Insert: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: store the VM name in a TaskRun annotation.
	// TODO: store the logs URL in a TaskRun annotation.

	// TODO: enqueue a Cloud Task to ensure the TaskRun has gotten some
	// update from the VM, or else cancel it and kill the VM (internal
	// error)

	// TODO: enqueue a Cloud Task to enforce TaskRun timeout.

	// Respond with TaskRun JSON
	if err := json.NewEncoder(w).Encode(tr); err != nil {
		log.Printf("json.Encode: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	name := strings.TrimPrefix(r.URL.Path, "/internal/taskruns/")

	defer r.Body.Close()
	var tr *v1beta1.TaskRun
	if err := json.NewDecoder(r.Body).Decode(tr); err != nil {
		log.Printf("json.Decode: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("updating %q", tr.Name)
	tr, err := s.storage.Update(ctx, name, tr.Status)
	if err != nil {
		log.Printf("Update: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with TaskRun JSON
	if err := json.NewEncoder(w).Encode(tr); err != nil {
		log.Printf("json.Encode: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	name := strings.TrimPrefix(r.URL.Path, "/apis/tekton.dev/v1beta1/default/taskruns/")
	tr, err := s.storage.Get(ctx, name)
	if err != nil {
		log.Printf("Get(%q): %v", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with TaskRun JSON
	if err := json.NewEncoder(w).Encode(tr); err != nil {
		log.Printf("json.Encode: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (s *server) delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	name := strings.TrimPrefix(r.URL.Path, "/apis/tekton.dev/v1beta1/default/taskruns/")
	if err := s.storage.Delete(ctx, name); err != nil {
		log.Printf("Delete(%q): %v", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
