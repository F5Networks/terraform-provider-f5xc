# Example: K8s Cluster - Basic

resource "f5xc_k8s_cluster" "example" {
  name      = "example-k8s-cluster"
  namespace = "system"

  local_access_choice = {
    no_local_access = true
  }

  global_access_choice = {
    no_global_access = true
  }

  pod_security_policy_choice = {
    use_default_psp = true
  }

  cluster_role_choice = {
    use_default_cluster_roles = true
  }

  cluster_role_bindings_choice = {
    use_default_cluster_role_bindings = true
  }

  pod_security_admission_choice = {
    use_default_pod_security_admission = true
  }

  apps_choice = {
    no_cluster_wide_apps = true
  }

  insecure_registries_choice = {
    no_insecure_registries = true
  }

  cluster_scoped_resource_access_choice = {
    cluster_scoped_access_deny = true
  }

  vk8s_namespace_access_choice = {
    vk8s_namespace_access_deny = true
  }
}
