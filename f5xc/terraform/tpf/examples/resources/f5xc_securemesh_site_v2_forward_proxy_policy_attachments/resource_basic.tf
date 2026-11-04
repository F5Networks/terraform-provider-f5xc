# Example: Securemesh Site V2 Forward Proxy Policy Attachments - Basic

resource "f5xc_securemesh_site_v2" "example" {
  name      = "example-securemesh-site-v2-0r6bkclb"
  namespace = "system"

  logs_receiver_choice = {
    logs_streaming_disabled = true
  }

  blocked_services_choice = {
    block_all_services = true
  }

  provider_choice = {
    kvm = {
      orchestration_choice = {
        not_managed = {}
      }
    }
  }
}

resource "f5xc_forward_proxy_policy" "fpp1" {
  name      = "tf-acc-fpp-fpp1-0r6bkclb"
  namespace = "system"

  proxy_choice = {
    any_proxy = true
  }

  rule_choice = {
    allow_all = true
  }
}

resource "f5xc_forward_proxy_policy" "fpp2" {
  name      = "tf-acc-fpp-fpp2-0r6bkclb"
  namespace = "system"

  proxy_choice = {
    any_proxy = true
  }

  rule_choice = {
    allow_all = true
  }
}

resource "f5xc_securemesh_site_v2_forward_proxy_policy_attachments" "example" {
  securemesh_site_v2_name = f5xc_securemesh_site_v2.example.name
  forward_proxy_policies = [
    f5xc_forward_proxy_policy.fpp1.name,
    f5xc_forward_proxy_policy.fpp2.name,
  ]
}
