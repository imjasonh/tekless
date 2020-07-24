## Prerequisites

Install [`gcloud`](https://cloud.google.com/sdk/install) and set your default
project: `gcloud config set project [MY-PROJECT]`.

Install [`ko`](https://github.com/google/ko) and set
`KO_DOCKER_REPO=gcr.io/$(gcloud config get-value project)`.

## Running a Pod directly

```
go run ./cmd/run/ \
  --project=$(gcloud config get-value project) \
  --tok=$(gcloud auth print-access-token) \
  --watcher_image=$(ko publish -P ./cmd/watcher) \
  --pod=pod.yaml
```

## Running a TaskRun

```
go run ./cmd/run/ \
  --project=$(gcloud config get-value project) \
  --tok=$(gcloud auth print-access-token) \
  --watcher_image=$(ko publish -P ./cmd/watcher) \
  --taskrun=taskrun.yaml
```

**VMs aren't killed when containers finish yet, so make sure to delete any VMs
yourself.**

### Debugging

SSH into the VM via Cloud Console.

See what containers are running:
```
docker ps
```

If containers aren't running, check the startup script status, and logs and
status of the `kubelet` that's supposed to be running your Pod:

```
sudo journalctl -u cloud-init.service
sudo journalctl -u kubelet.service
systemctl status kubelet.service
```

If Pods are running, you can ask the kubelet for details:

```
curl -v  --insecure https://localhost:10250/pods
```

And for logs:

```
curl -v --insecure https://localhost:10250/logs/
curl -v --insecure https://localhost:10250/logs/containers/[CONTAINER-NAME]
curl -v --insecure https://localhost:10250/logs/pods/[POD-NAME]/[CONTAINER-NAME]/0.log
```

In all of these cases, omitting a path segment will make kubelet respond with
known valid values, which can be very helpful since pod names and container
names are generated and not known ahead of time.

**BUG:** Currently, TaskRun execution fails to run the first step, due to some
permissions issue with the entrypoint binary running the step:

```
{"log":"standard_init_linux.go:211: exec user process caused \"permission denied\"\n","stream":"stderr","time":"2020-07-21T13:31:12.681426353Z"}
```

## Deploying the API Service

```
ko resolve -P -f config/service.yaml | \
    sed -e 's/---//g' | \
    gcloud beta run services replace - --platform=managed --region=us-east4
```

You can also set gcloud defaults:

```
gcloud config set run/region us-east4
gcloud config set run/platform managed
```

### Running a TaskRun

```
curl -X POST -H "Content-Type: application/json" -d @taskrun.json https://api-zrxvguzx2q-uk.a.run.app/apis/tekton.dev/v1beta1/default/taskruns
```
