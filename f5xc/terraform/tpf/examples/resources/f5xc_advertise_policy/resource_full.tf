# Example: Advertise Policy - Full

resource "f5xc_virtual_network" "example" {
  name      = "example-vn-4q6hl815"
  namespace = "system"

  network_choice = {
    site_local_network = true
  }
}

resource "f5xc_advertise_policy" "example" {
  depends_on = [f5xc_virtual_network.example]

  name            = "example-advertise-policy-4q6hl815"
  namespace       = "default"
  description     = "Acceptance test advertise policy"
  protocol        = "UDP"
  skip_xff_append = true

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "networking"
  }

  where = {
    ref_or_selector_choice = {
      virtual_network = {
        ref = [{
          namespace = f5xc_virtual_network.example.namespace
          name      = f5xc_virtual_network.example.name
        }]
      }
    }
  }

  port_choice = {
    port_ranges = "443,8443"
  }

  tls_parameters = {
    client_certificate_verify_choice = {
      no_client_certificate = true
    }

    xfcc_header_elements = ["XFCC_SUBJECT"]

    common_params = {
      minimum_protocol_version = ["TLSv1_2"]
      maximum_protocol_version = ["TLSv1_3"]
      cipher_suites            = ["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"]
    }
  }
}
