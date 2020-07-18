package pkg

var trueString = "true"

var cloudConfig = `#cloud-config

runcmd:
- systemctl daemon-reload
- systemctl enable kubelet.service
- systemctl start kubelet.service

write_files:
- path: /etc/kubernetes/kubelet-config.yaml
  content: |
    apiVersion: kubelet.config.k8s.io/v1beta1
    kind: KubeletConfiguration
    authentication:
      webhook:
        enabled: false
      anonymous:
        enabled: true
    authorization:
      mode: AlwaysAllow
    healthzPort: 10248
    imageGCHighThresholdPercent: 100
    port: 10250
    staticPodPath: /etc/kubernetes/manifests

- path: /etc/cni/net.d/dummy.conf
  content: |
    {"type":"bridge"}

- path: /etc/systemd/system/kubelet.service
  content: |
    [Service]
    ExecStartPre=/bin/mkdir -p /etc/certs
    ExecStartPre=/bin/bash -c "/usr/share/google/get_metadata_value attributes/ca-cert > /etc/certs/cacert.pem"
    ExecStartPre=/bin/bash -c "/usr/share/google/get_metadata_value attributes/ca-cert-key > /etc/certs/key.pem"
    ExecStartPre=/bin/mkdir -p /etc/kubernetes/manifests
    ExecStartPre=/bin/bash -c "/usr/share/google/get_metadata_value attributes/pod > /etc/kubernetes/manifests/pod.yaml"
    ExecStart=/usr/bin/kubelet \
      --config=/etc/kubernetes/kubelet-config.yaml \
      --container-runtime=remote \
      --container-runtime-endpoint=unix:///var/run/containerd/containerd.sock
    Restart=always
    RestartSec=10s
    
    [Install]
    WantedBy=multi-user.target
`

// TODO: generate these.
var caCert = `
-----BEGIN CERTIFICATE-----
MIIDCzCCAfOgAwIBAgIQRSWuCXPLy9BFdxryczX+jzANBgkqhkiG9w0BAQsFADAc
MRowGAYDVQQKExFBcmdvIEluY29ycG9yYXRlZDAeFw0xOTA1MDgxOTE0MDZaFw0y
MDA1MDcxOTE0MDZaMBwxGjAYBgNVBAoTEUFyZ28gSW5jb3Jwb3JhdGVkMIIBIjAN
BgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAl8fLJ5YJ6lm/0cdJb+wGr+lY2qSR
KdXZCu/ZAiy5wbSy4sEHjaZhNtg92zx/clz6k98DyNBzupRT5f6T409ugNNc9Bvs
sZkSOOYDsv4r8zl1lFCblN6wtbPXhF1wNiHBgJw6OJtQuotwvYh+6XC8Uq9DtX8/
oE7uXWCtmgwbwHvKm5K3Jlhd47S/u2h9OCIL1DLAN1utc092kE9Xp94/L5B/oWQw
rf9HI6/RjrJoQgVkhquQpee9rUwfsxKbiqbFA4xXnc59ZnNTfQOaQCrIstdlQ5Ac
WZsbDprX88k4zNWE144SM8ZtaSJGwpz4bLdHY2EAJAX7FIOHZL5yP5+GiwIDAQAB
o0kwRzAOBgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0T
AQH/BAUwAwEB/zAPBgNVHREECDAGhwQj3S46MA0GCSqGSIb3DQEBCwUAA4IBAQAc
jUNgMl/izQZTefr2/fExrAttNsJ+LeVIqU+YPW78rl63QVVeLLNtBvzm82kTbLzt
anv1DWt7oBT7pXelGAiueG1vbyZaV4w91EJCvmPmW3oEOgnV64A5d62osZkX9uwK
AvhmIJttgUfc5Cs7jGNEwC9rq3+aMJycksbOLcpEi8X3a8O9RmjFNVTYWWDlXYSB
tm0ktkbWX5o8dsBr4E23tOAYpTwOng44ZR2yYp50GONntZ40uZKbVE/ynh93tArF
7lLKc6ZPppyMsK/areed90AryY0Uo0+8jHYtgdywIN53nnUjxSuk2BoRxbPdy8XT
XnelKSwSRVEi7EFbAcrF
-----END CERTIFICATE-----`

var caCertKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAl8fLJ5YJ6lm/0cdJb+wGr+lY2qSRKdXZCu/ZAiy5wbSy4sEH
jaZhNtg92zx/clz6k98DyNBzupRT5f6T409ugNNc9BvssZkSOOYDsv4r8zl1lFCb
lN6wtbPXhF1wNiHBgJw6OJtQuotwvYh+6XC8Uq9DtX8/oE7uXWCtmgwbwHvKm5K3
Jlhd47S/u2h9OCIL1DLAN1utc092kE9Xp94/L5B/oWQwrf9HI6/RjrJoQgVkhquQ
pee9rUwfsxKbiqbFA4xXnc59ZnNTfQOaQCrIstdlQ5AcWZsbDprX88k4zNWE144S
M8ZtaSJGwpz4bLdHY2EAJAX7FIOHZL5yP5+GiwIDAQABAoIBAD39aUFBO8p9nmTg
1mMCTJbjIJmn9evWvd8EJ6cGOlXdZeRLvziAuBqsxdK5SjoctHDZeFO3o1SUSRHZ
4G/J3NF7we6nSwwb/v/DHcaona2ojZemNmzeaODFU2Ppv680KTJMXFELSjTuR3z5
dxADrb69e1Nw5b0lD6COoEiW4mzB4Twv7TClZE6f2UwTZBBA+CA0ltA757vwp5pl
Ff3X/7Th3C4s5CqeJMnPV8POtAUffuq0VAYjULarXW6bBTkERZpI+9XIv8pyHdSi
+ITjyim4+NCZvguL0n1CgWpRXzKrTd89/+UrzbiX+h3gtZUktCSI534dHqPJIRAQ
NXrnyt0CgYEAz2XrRYkKkfdt/YcP9L/9YzqjQ6tdxHmYltliVu4OjozZa5ahusQe
TzFksAckhn6MdDAeXbCKeP06eFfrXwt8TiJj0/dgJJI5UKF+nNFMgdJVaKkVLA/z
ODICDLEjQe75i818J7kH/vGhYZGpHrnbqE9yeb/M+sNfuKxWQTmbnnUCgYEAu1lM
XEHGFfzoM/T5+bBYzIfl7XG8bKDIYrDhIUYA2q+LfSb1fesyF8p3iDMfMAPCDBFL
fajwOnWBs7t/pjz1MgkxcjFiNZVFbGLqukPcI4XXurhEcfrvJ0dEL/zCjmujjGds
KsmYDmQDugZNfIQhGQYZ+Y7+HyNy+zm4zFDz8P8CgYBB7mKGrnQfxwq+SAt4gPgq
bV+tiXK7nPQ3MFAk1nTmODx+CVrMpsAD6O3bT8n6v4wi+5ELs62xnL7Ttw9qHZqc
tC4MGl4EAkAaM9yuOZMayiTAqs/CPCfTu4IPStisgy3tlZtcfWPfVi05eTbMP8Vm
kisQLTsalLV/Xbnl7kxcaQKBgBxBT37qfJF8XxjW7Yx2yC2woUC6Uoyfgxk/S/TU
tfRFXWg2o/elrRxhcL2d2CpJps5jHVuKSxDGABW3RX0w3Fn7gPWT9RfXt2ytTnFh
IqZI3UxP1iLKkZ7+5I3INR99pGDciDe7x68D7nvzz2PkGYnIncpBgpn3orO49OH7
o47JAoGBAMuX4ATyS7NnHklbcXp3djgLy5ZlVEsbNUmDG/zUtp95T0AzkYSTkukn
zPHXoKanMiyZ7HQ+gHoJoOxoRYEGe1ch9AMPlcnbD4Svaeaz3s+JZ2r3aYielNjl
pYblimIhRZ3ZwpDJDO3yA7zONLZ21xzaWJ0Ut2zUFlFW6+Jl+TlJ
-----END RSA PRIVATE KEY-----`
