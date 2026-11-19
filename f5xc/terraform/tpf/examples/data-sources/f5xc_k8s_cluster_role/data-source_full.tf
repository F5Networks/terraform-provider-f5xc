# Example: K8s Cluster Role - Full

data "f5xc_k8s_cluster_role" "example" {
  name      = f5xc_k8s_cluster_role.example.name
  namespace = f5xc_k8s_cluster_role.example.namespace
}
