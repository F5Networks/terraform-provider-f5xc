# Example: Virtual K8s - Vsite Refs

resource "f5xc_namespace" "test_namespace" {
  name = "example-namespace"
}

resource "f5xc_virtual_site" "test_vsite_a" {
  name      = "example-virtual-site-a"
  namespace = f5xc_namespace.test_namespace.name

  site_selector = {
    expressions = ["environment in (example)"]
  }

  site_type = ["REGIONAL_EDGE"]
}

resource "f5xc_virtual_k8s" "example" {
  name      = "example-virtual-k8s"
  namespace = f5xc_namespace.test_namespace.name

  vsite_refs = [{
    name      = f5xc_virtual_site.test_vsite_a.name
    namespace = f5xc_virtual_site.test_vsite_a.namespace
  }]
}
