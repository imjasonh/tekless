## Prerequisites

Install [`ko`](https://github.com/google/ko) and set
`KO_DOCKER_REPO=gcr.io/$(gcloud config get-value project)`.

## Running a Pod directly

```
go run ./cmd/run/ \
  --project=$(gcloud config get-value project) \
  --tok=$(gcloud auth print-access-token) \
  --watcher_image=$(ko publish ./cmd/watcher) \
  -f=pod.yaml
```

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

VMs aren't killed when containers finish yet, so make sure to delete any VMs
yourself.

## Deploying the API Service

```
ko resolve -f config/service.yaml | \
    sed -e 's/---//g' | \
    gcloud beta run services replace - --platform=managed --region=us-east4
```

You can also set gcloud defaults:

```
gcloud config set run/region us-east4
gcloud config set run/platform managed
```
