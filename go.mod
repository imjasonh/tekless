module github.com/imjasonh/tekless

go 1.14

require (
	github.com/ghodss/yaml v1.0.0
	github.com/google/uuid v1.1.1
	github.com/tektoncd/pipeline v0.14.2
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.29.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.18.6
)

// Knative deps (release-0.15)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.4.0+incompatible
	knative.dev/caching => knative.dev/caching v0.0.0-20200521155757-e78d17bc250e
	knative.dev/pkg => knative.dev/pkg v0.0.0-20200528142800-1c6815d7e4c9
)

// Pin k8s deps to 1.16.5
replace (
	k8s.io/api => k8s.io/api v0.16.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.5
	k8s.io/client-go => k8s.io/client-go v0.16.5
	k8s.io/code-generator => k8s.io/code-generator v0.16.5
)
