package main

import (
	"flag"
	"log"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	client "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

var (
	addr = flag.String("host", "localhost:8080", "API server address")
	// TODO: take taskrun.yaml as a flag.
)

func main() {
	flag.Parse()

	c := client.NewForConfigOrDie(&rest.Config{
		Host: *addr,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true, // TODO: use TLS
		},
	})
	tr, err := c.TaskRuns("namespace").Create(&v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-run",
		},
		Spec: v1beta1.TaskRunSpec{
			TaskSpec: &v1beta1.TaskSpec{
				Steps: []v1beta1.Step{{Container: corev1.Container{
					Image:   "busybox",
					Command: []string{"echo", "hello"},
				}}},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Println(tr)
}
