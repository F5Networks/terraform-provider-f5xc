# Example: K8s Cluster Role - Basic

resource "f5xc_k8s_cluster_role" "example" {
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
