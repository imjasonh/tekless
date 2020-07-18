## Running a Pod directly

```
go run ./cmd/run/ \
  --project=$(gcloud config get-value project) \
  --tok=$(gcloud auth print-access-token) \
  -f=pod.yaml
```

## Debugging

SSH into the VM:

```
sudo journalctl -u cloud-init.service
sudo journalctl -u kubelet.service
systemctl status kubelet.service
```

kubelet currently fails to start pod with:

```
failed to find plugin "loopback" in path [/opt/cni/bin]
```
