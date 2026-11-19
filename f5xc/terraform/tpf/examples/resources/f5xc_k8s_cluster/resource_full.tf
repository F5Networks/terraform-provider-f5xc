# Example: K8s Cluster - Full

resource "f5xc_k8s_cluster_role" "example_role" {
  name      = "example-k8s-cluster-role"
  namespace = "system"

  rule_choice = {
    policy_rule_list = {
      policy_rule = [{
        resource_choice = {
          resource_list = {
            api_groups     = ["apps"]
            resource_types = ["deployments"]
            verbs          = ["get", "list"]
          }
        }
      }]
    }
  }
}

resource "f5xc_k8s_cluster_role_binding" "example_crb" {
  name      = "example-k8s-cluster-role-binding"
  namespace = "system"

  k8s_cluster_role = {
    name      = f5xc_k8s_cluster_role.example_role.name
    namespace = f5xc_k8s_cluster_role.example_role.namespace
  }

  subjects = [
    {
      subject_choice = {
        user = "user1@example.com"
      }
    },
  ]
}

resource "f5xc_k8s_pod_security_admission" "example_psa" {
  name      = "example-k8s-pod-security-admission"
  namespace = "system"

  pod_security_admission_specs = [
    {
      policy_type_choice = {
        baseline = true
      }
      admission_mode_choice = {
        enforce = true
      }
    },
  ]
}

resource "f5xc_k8s_cluster" "example" {
  name        = "example-k8s-cluster"
  namespace   = "system"
  description = "K8s cluster full example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "example-team"
  }

  local_access_choice = {
    local_access_config = {
      local_domain = "example.com"
      port_choice = {
        port = 443
      }
    }
  }

  global_access_choice = {
    global_access_enable = true
  }

  pod_security_policy_choice = {
    use_default_psp = true
  }

  cluster_role_choice = {
    use_custom_cluster_role_list = {
      cluster_roles = [
        {
          name      = f5xc_k8s_cluster_role.example_role.name
          namespace = f5xc_k8s_cluster_role.example_role.namespace
        },
      ]
    }
  }

  cluster_role_bindings_choice = {
    use_custom_cluster_role_bindings = {
      cluster_role_bindings = [
        {
          name      = f5xc_k8s_cluster_role_binding.example_crb.name
          namespace = f5xc_k8s_cluster_role_binding.example_crb.namespace
        },
      ]
    }
  }

  pod_security_admission_choice = {
    use_custom_pod_security_admission = {
      name      = f5xc_k8s_pod_security_admission.example_psa.name
      namespace = f5xc_k8s_pod_security_admission.example_psa.namespace
    }
  }

  apps_choice = {
    cluster_wide_app_list = {
      cluster_wide_apps = [
        {
          app_choice = {
            dashboard = true
          }
        },
        {
          app_choice = {
            metrics_server = true
          }
        },
        {
          app_choice = {
            prometheus = true
          }
        },
      ]
    }
  }

  insecure_registries_choice = {
    insecure_registry_list = {
      insecure_registries = ["example.com:5000"]
    }
  }

  cluster_scoped_resource_access_choice = {
    cluster_scoped_access_permit = true
  }

  vk8s_namespace_access_choice = {
    vk8s_namespace_access_permit = true
  }
}
