## Running a Pod directly

```
go run ./cmd/run/ \
  --project=$(gcloud config get-value project) \
  --tok=$(gcloud auth print-access-token) \
  -f=pod.yaml
```

## Debugging

SSH into the VM via Cloud Console.

```
sudo journalctl -u cloud-init.service
sudo journalctl -u kubelet.service
systemctl status kubelet.service
```

kubelet currently fails to start pod with:

```
failed to find plugin "loopback" in path [/opt/cni/bin]
```


## Deploying the API Service

Install [`ko`](https://github.com/google/ko) and set
`KO_DOCKER_REPO=gcr.io/$(gcloud config get-value project)`.

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
