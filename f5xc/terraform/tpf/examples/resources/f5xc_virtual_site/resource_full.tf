# Example: Virtual Site - Full

resource "f5xc_virtual_site" "example" {
  name        = "example-virtual-site"
  namespace   = "default"
  description = "Virtual site full example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "example-team"
  }

  site_selector = {
    expressions = ["environment in (example)"]
  }

  site_type = ["REGIONAL_EDGE"]
}
