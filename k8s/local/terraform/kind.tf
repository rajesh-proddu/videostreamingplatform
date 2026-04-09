terraform {
  required_version = ">= 1.0"
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "~> 0.7.0"
    }
  }
}

provider "kind" {}

resource "kind_cluster" "default" {
  name           = "videostreamingplatform"
  node_image     = "kindest/node:v1.32.2"
  wait_for_ready = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"

    node {
      role = "control-plane"
      extra_port_mappings {
        container_port = 30080
        host_port      = 8080
        protocol       = "TCP"
      }
      extra_port_mappings {
        container_port = 30081
        host_port      = 8081
        protocol       = "TCP"
      }
      extra_port_mappings {
        container_port = 30090
        host_port      = 9090
        protocol       = "TCP"
      }
      extra_port_mappings {
        container_port = 30686
        host_port      = 16686
        protocol       = "TCP"
      }
      extra_port_mappings {
        container_port = 30300
        host_port      = 3000
        protocol       = "TCP"
      }
    }

    node {
      role = "worker"
    }
  }
}

output "cluster_name" {
  value       = kind_cluster.default.name
  description = "Kind cluster name"
}

output "kubeconfig" {
  value       = kind_cluster.default.kubeconfig
  description = "Kubeconfig for the cluster"
  sensitive   = true
}
