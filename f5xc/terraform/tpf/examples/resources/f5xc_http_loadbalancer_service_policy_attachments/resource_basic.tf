# Example: HTTP LoadBalancer Service Policy Attachments

resource "f5xc_origin_pool" "example" {
  name                   = "example-origin-pool"
  namespace              = "default"
  loadbalancer_algorithm = ["ROUND_ROBIN"]
  endpoint_selection     = ["LOCAL_PREFERRED"]

  port_choice = {
    port = 80
  }

  origin_servers = [
    {
      choice = {
        public_name = {
          dns_name = "example.com"
        }
      }
    }
  ]

  tls_choice = {
    no_tls = true
  }

  upstream_conn_pool_reuse_type = {
    map_downstream_to_upstream_conn_pool_type_choice = {
      disable_conn_pool_reuse = true
    }
  }
}

resource "f5xc_service_policy" "allow_all" {
  name          = "example-service-policy"
  namespace     = "default"
  rule_choice   = { allow_all_requests = true }
  server_choice = { any_server = true }
}

resource "f5xc_service_policy" "deny_all" {
  name          = "example-deny-policy"
  namespace     = "default"
  rule_choice   = { deny_all_requests = true }
  server_choice = { any_server = true }
}

# service_policy_choice is omitted so the attachment resource below owns the policy list.
resource "f5xc_http_loadbalancer" "example" {
  name      = "example-http-loadbalancer"
  namespace = "default"
  domains   = ["example.com"]

  loadbalancer_type_choice = {
    http = {
      dns_volterra_managed = false
      port_choice = {
        port = 80
      }
    }
  }

  default_route_pools = [
    {
      pool_choice = {
        pool = {
          name      = f5xc_origin_pool.example.name
          namespace = "default"
        }
      }
      weight   = 1
      priority = 1
    }
  ]

  advertise_choice                = { do_not_advertise = true }
  api_definition_choice           = { disable_api_definition = true }
  api_discovery_choice            = { disable_api_discovery = true }
  api_testing_choice              = { disable_api_testing = true }
  challenge_type_choice           = { no_challenge = true }
  hash_policy_choice              = { round_robin = true }
  malicious_user_detection_choice = { disable_malicious_user_detection = true }
  malware_protection_choice       = { disable_malware_protection = true }
  rate_limit_choice               = { disable_rate_limit = true }
  sensitive_data_policy_choice    = { default_sensitive_data_policy = true }
  threat_mesh_choice              = { disable_threat_mesh = true }
  trust_client_ip_headers_choice  = { disable_trust_client_ip_headers = true }
  user_id_choice                  = { user_id_client_ip = true }
  waf_choice                      = { disable_waf = true }
}

# Authoritatively manages the complete ordered service policy list on the load balancer.
resource "f5xc_http_loadbalancer_service_policy_attachments" "example" {
  namespace              = "default"
  http_loadbalancer_name = f5xc_http_loadbalancer.example.name
  service_policies = [
    f5xc_service_policy.allow_all.name,
    f5xc_service_policy.deny_all.name,
  ]
}
