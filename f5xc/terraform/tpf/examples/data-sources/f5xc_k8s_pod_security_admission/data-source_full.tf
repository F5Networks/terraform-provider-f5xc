# Example: K8s Pod Security Admission - Full

data "f5xc_k8s_pod_security_admission" "example" {
  name      = f5xc_k8s_pod_security_admission.example.name
  namespace = f5xc_k8s_pod_security_admission.example.namespace
}
