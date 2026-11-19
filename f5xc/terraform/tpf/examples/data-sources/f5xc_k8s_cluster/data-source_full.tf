# Example: K8s Cluster - Full

data "f5xc_k8s_cluster" "example" {
  name      = f5xc_k8s_cluster.example.name
  namespace = f5xc_k8s_cluster.example.namespace
}
