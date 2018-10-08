package main

import (
	"encoding/base64"
	"strings"
)

// k8s node setup steps from
// from https://kubernetes.io/docs/setup/independent/install-kubeadm/

var hashBang = "#!/bin/bash"
var disableSwap = `swapoff -a`
var installBase = `
CNI_VERSION="v0.6.0"
mkdir -p /opt/cni/bin
curl -L "https://github.com/containernetworking/plugins/releases/download/${CNI_VERSION}/cni-plugins-amd64-${CNI_VERSION}.tgz" | tar -C /opt/cni/bin -xz

# install crictl
CRICTL_VERSION="v1.11.1"
mkdir -p /opt/bin
curl -L "https://github.com/kubernetes-incubator/cri-tools/releases/download/${CRICTL_VERSION}/crictl-${CRICTL_VERSION}-linux-amd64.tar.gz" | tar -C /opt/bin -xz

# install kubernetes components
RELEASE="$(curl -sSL https://dl.k8s.io/release/stable.txt)"

mkdir -p /opt/bin
cd /opt/bin
curl -L --remote-name-all https://storage.googleapis.com/kubernetes-release/release/${RELEASE}/bin/linux/amd64/{kubeadm,kubelet,kubectl}
chmod +x {kubeadm,kubelet,kubectl}

curl -sSL "https://raw.githubusercontent.com/kubernetes/kubernetes/${RELEASE}/build/debs/kubelet.service" | sed "s:/usr/bin:/opt/bin:g" > /etc/systemd/system/kubelet.service
mkdir -p /etc/systemd/system/kubelet.service.d
curl -sSL "https://raw.githubusercontent.com/kubernetes/kubernetes/${RELEASE}/build/debs/10-kubeadm.conf" | sed "s:/usr/bin:/opt/bin:g" > /etc/systemd/system/kubelet.service.d/10-kubeadm.conf`

func masterData() string {
	cmd := []string{hashBang, disableSwap, installBase}
	return base64.StdEncoding.EncodeToString([]byte(strings.Join(cmd, "\n")))
}

func nodeData() string {
	cmd := []string{hashBang, disableSwap, installBase}
	return base64.StdEncoding.EncodeToString([]byte(strings.Join(cmd, "\n")))
}
