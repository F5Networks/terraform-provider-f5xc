# Example: K8s Cluster Role - Full

resource "f5xc_k8s_cluster_role" "example" {
  name        = "example-k8s-cluster-role"
  namespace   = "system"
  description = "K8s cluster role full example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "example-team"
  }

  rule_choice = {
    policy_rule_list = {
      policy_rule = [
        {
          resource_choice = {
            resource_list = {
              api_groups         = ["rbac.authorization.k8s.io"]
              resource_types     = ["roles"]
              resource_instances = ["admin"]
              verbs              = ["get", "watch"]
            }
          }
        },
        {
          resource_choice = {
            non_resource_url_list = {
              urls  = ["/healthz"]
              verbs = ["get"]
            }
          }
        },
      ]
    }
  }
}
