# Example: K8s Pod Security Admission - Basic

resource "f5xc_k8s_pod_security_admission" "example" {
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
