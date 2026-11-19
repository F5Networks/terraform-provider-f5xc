# Example: K8s Cluster Role Binding - Full

data "f5xc_k8s_cluster_role_binding" "example" {
  name      = f5xc_k8s_cluster_role_binding.example.name
  namespace = f5xc_k8s_cluster_role_binding.example.namespace
}
