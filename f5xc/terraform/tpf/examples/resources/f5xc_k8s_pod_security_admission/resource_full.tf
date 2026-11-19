# Example: K8s Pod Security Admission - Full

resource "f5xc_k8s_pod_security_admission" "example" {
  name        = "example-k8s-pod-security-admission"
  namespace   = "system"
  description = "K8s pod security admission full example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "example-team"
  }

  pod_security_admission_specs = [
    {
      policy_type_choice = {
        restricted = true
      }
      admission_mode_choice = {
        enforce = true
      }
    },
    {
      policy_type_choice = {
        baseline = true
      }
      admission_mode_choice = {
        audit = true
      }
    },
    {
      policy_type_choice = {
        privileged = true
      }
      admission_mode_choice = {
        warn = true
      }
    },
  ]
}
