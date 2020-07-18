package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
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

	svc, err := compute.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *tok})))
	if err != nil {
		log.Fatal("compute.NewService: %v", err)
	}

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
		// TODO: enable secure boot
	}).Do()
	if err != nil {
		log.Fatalf("instances.insert: %v", err)
	}

	start := time.Now()
	for ; ; time.Sleep(time.Second) {
		op, err = svc.ZoneOperations.Get(*project, *zone, op.Name).Do()
		if err != nil {
			log.Fatalf("operations.get(%s): %v", op.Name, err)
		}
		log.Printf("operation is %q...", op.Status)
		if op.Status == "DONE" {
			break
		}
	}
	log.Println("startup took", time.Since(start))
}
