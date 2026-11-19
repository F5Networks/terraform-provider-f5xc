# Example: K8s Cluster Role Binding - Full

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

resource "f5xc_k8s_cluster_role_binding" "example" {
  name        = "example-k8s-cluster-role-binding"
  namespace   = "system"
  description = "K8s cluster role binding full example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "example-team"
  }

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
    {
      subject_choice = {
        group = "group1"
      }
    },
    {
      subject_choice = {
        service_account = {
          namespace = "cd-app-namespace"
          name      = "cd-app-controller"
        }
      }
    },
  ]
}
