package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	corev1 "k8s.io/api/core/v1"
)

func RunPod(ctx context.Context, pod corev1.Pod,
	tok, project, zone, machineType string) error {

	svc, err := compute.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok})))
	if err != nil {
		return err
	}

	b, err := json.Marshal(pod)
	if err != nil {
		return err
	}
	podstr := string(b)

	region := zone[:strings.LastIndex(zone, "-")]

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
