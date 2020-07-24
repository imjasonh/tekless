package pkg

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// Support images from v0.14.2
var images = pipeline.Images{
	EntrypointImage: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/entrypoint:v0.14.2@sha256:3db5b1622b939b11603b49916cdfb5718e25add7a9c286a2832afb16f57f552f",
	ShellImage:      "gcr.io/distroless/base@sha256:f79e093f9ba639c957ee857b1ad57ae5046c328998bf8f72b30081db4d8edbe4",
}

func RunTaskRun(ctx context.Context, tr v1beta1.TaskRun,
	ts oauth2.TokenSource, watcherImage, project, zone, machineType string) error {

	// Validate TaskRun request (must inline taskSpec)
	// TODO: support referencing Tasks
	if err := tr.Validate(ctx); err != nil {
		return err
	}
	if tr.Spec.TaskSpec == nil {
		return errors.New("must inline taskSpec")
	}

	saName := tr.Spec.ServiceAccountName
	if saName == "" {
		saName = "default"
	}
	kubeclient := fake.NewSimpleClientset(
		// Pretend there's a ServiceAccount with no Secrets to add.
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: saName,
			},
		},
	)

	// MakePod assumes TaskRun already has annotations...
	if tr.Annotations == nil {
		tr.Annotations = map[string]string{}
	}
	// TODO: MakePod modifies TaskRun when using script mode.
	p, err := (&pod.Builder{
		Images:          images,
		KubeClient:      kubeclient,
		EntrypointCache: nil,
		OverrideHomeEnv: false,
	}).Build(ctx, &tr, *tr.Spec.TaskSpec)
	if err != nil {
		return err
	}

	// TODO: do this in pod.go
	p.TypeMeta = metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Pod",
	}
	// TODO: provide downward API value since there aren't sidecars to wait for.
	p.ObjectMeta.Annotations["tekton.dev/ready"] = "READY"
	// TODO: the TaskRun owner has uid:"" which kubelet doesn't like...
	p.ObjectMeta.OwnerReferences = nil

	return RunPod(ctx, *p, ts, watcherImage, project, zone, machineType)
}

// RunPod runs the Pod on a new VM.
func RunPod(ctx context.Context, pod corev1.Pod,
	ts oauth2.TokenSource, watcherImage, project, zone, machineType string) error {

	svc, err := compute.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return err
	}

	b, err := yaml.Marshal(pod)
	if err != nil {
		return err
	}
	podstr := string(b)
	log.Println("POD MANIFEST:\n", podstr) // TODO remove

	region := zone[:strings.LastIndex(zone, "-")]

	watcherPod := fmt.Sprintf(watcherPodFmt, watcherImage)

	name := "instance-" + uuid.New().String()[:4]
	log.Printf("creating %q...", name)
	op, err := svc.Instances.Insert(project, zone, &compute.Instance{
		Name:        name,
		Zone:        zone,
		MachineType: fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", project, zone, machineType),
		Disks: []*compute.AttachedDisk{{
			InitializeParams: &compute.AttachedDiskInitializeParams{
				SourceImage: "projects/cos-cloud/global/images/family/cos-stable",
			},
			Boot: true,
		}},
		NetworkInterfaces: []*compute.NetworkInterface{{
			Subnetwork: fmt.Sprintf("projects/%s/regions/%s/subnetworks/default", project, region),
			AccessConfigs: []*compute.AccessConfig{{
				Name:        "External NAT",
				Type:        "ONE_TO_ONE_NAT",
				NetworkTier: "PREMIUM",
			}},
		}},
		ServiceAccounts: []*compute.ServiceAccount{{
			Email: "178371766757-compute@developer.gserviceaccount.com",
			Scopes: []string{
				// Permiission to pull private images (watcher)
				"https://www.googleapis.com/auth/devstorage.read_only",

				// Permission to write logs and metrics (google-fluentd)
				"https://www.googleapis.com/auth/logging.write",
				"https://www.googleapis.com/auth/monitoring.write",
			},
		}},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{{
				Key:   "user-data",
				Value: &cloudConfig,
			}, {
				Key:   "watcher",
				Value: &watcherPod,
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
		ShieldedInstanceConfig: &compute.ShieldedInstanceConfig{
			EnableSecureBoot: true,
		},
	}).Do()
	if err != nil {
		return err
	}

	start := time.Now()
	for ; ; time.Sleep(time.Second) {
		op, err = svc.ZoneOperations.Get(project, zone, op.Name).Do()
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
