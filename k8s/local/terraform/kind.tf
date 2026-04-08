terraform {
  required_version = ">= 1.0"
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "~> 0.2.1"
    }
  }
}

provider "kind" {}

resource "kind_cluster" "default" {
  name           = "videostreamingplatform"
  node_image     = "kindest/node:v1.28.0"
  wait_for_ready = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"

    node {
      role = "control-plane"
    }

    node {
      role = "worker"
    }

    node {
      role = "worker"
    }

    containerd_config_patches = [
      "[[plugins.\"io.containerd.grpc.v1.cri\".registry.mirrors.\"localhost:5000\"]]\n  endpoint = [\"http://localhost:5000\"]"
    ]
  }
}

output "cluster_name" {
  value       = kind_cluster.default.name
  description = "Kind cluster name"
}

output "cluster_context" {
  value       = kind_cluster.default.kubeconfig_context_name
  description = "Kubernetes context name"
}
