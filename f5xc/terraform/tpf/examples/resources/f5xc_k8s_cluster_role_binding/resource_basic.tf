# Example: K8s Cluster Role Binding - Basic

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
