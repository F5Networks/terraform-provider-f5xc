terraform {
  required_providers {
    f5xc = {
      source  = "localhost/f5/f5xc"
      version = "0.1.1"
    }
  }
}

variable "f5xc_api_url" {
  type = string
}

variable "f5xc_api_p12_file" {
  type = string
}

variable "f5xc_api_token" {
  type = string
}

provider "f5xc" {
  url       = var.f5xc_api_url
  timeout   = "30s"
  api_token = var.f5xc_api_token
  #api_p12_file = var.f5xc_api_p12_file
  limiter = {
    rate  = 30
    burst = 1
  }
}

resource "f5xc_healthcheck" "hc1" {
  name        = "test-hc-1"
  namespace   = "default"
  description = "healthcheck test1"
  health_check_choice = {
    http_health_check = {
      host_header_choice = {
        use_origin_server_name = true
      }
    }
  }
}

resource "f5xc_origin_pool" "op" {
  name                   = "test-op-123"
  namespace              = "default"
  description            = "origin pool test"
  loadbalancer_algorithm = ["ROUND_ROBIN"]
  tls_choice = {
    no_tls = true
  }
  port_choice = {
    port = 8080
  }
  endpoint_selection = ["LOCAL_PREFERRED"]
  origin_servers = [
    {
      choice = {
        private_ip = {
          private_ip_choice = {
            ip = "192.168.1.1"
          }
          site_locator = {
            choice = {
              site = {
                name      = "test-site-1"
                namespace = "default"
              }
            }
          }
          network_choice = {
            inside_network = true
          }
        }
      }
    }
  ]
  healthcheck = [
    {
      name      = f5xc_healthcheck.hc1.name
      namespace = f5xc_healthcheck.hc1.namespace
    }
  ]
}

resource "f5xc_alert_policy" "ap1" {
  name      = "test-alert-policy-1"
  namespace = "default"
  receivers = [
    {
      namespace = "default"
      name      = "ck-test-01"
    }
  ]
  routes = [
    {
      action_choice = {
        send = true
      }
    }
  ]
}

resource "f5xc_token" "t1" {
  name      = "ck-test-token-01"
  namespace = "system"
}

resource "f5xc_namespace" "ns1" {
  name = "ck-test-ns1"
}

resource "f5xc_virtual_k8s" "vk8s1" {
  name      = "test-vk8s-1"
  namespace = f5xc_namespace.ns1.name
}



/*
resource "f5xc_api_definition_uploader" "swagger" {
  metadata = {
    name = "test126"
    namespace = "default"
  }
  string_value = file("/Users/c.klewar/Projects/gitlab.com/F5/volterra/ves.io/terraform-provider-f5xc/examples/tpf/openapi.json")
}
*/

/*
resource "f5xc_token" "token" {
  name = "token123"
}
*/


