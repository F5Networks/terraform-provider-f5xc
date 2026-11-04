# Example: Securemesh Site V2 Enhanced Firewall Policy Attachments - Basic

resource "f5xc_securemesh_site_v2" "example" {
  name      = "example-securemesh-site-v2-wshqanjn"
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

resource "f5xc_enhanced_firewall_policy" "efp1" {
  name      = "tf-acc-efp-efp1-wshqanjn"
  namespace = "system"

  rule_choice = {
    allow_all = true
  }
}

resource "f5xc_enhanced_firewall_policy" "efp2" {
  name      = "tf-acc-efp-efp2-wshqanjn"
  namespace = "system"

  rule_choice = {
    allow_all = true
  }
}

resource "f5xc_securemesh_site_v2_enhanced_firewall_policy_attachments" "example" {
  securemesh_site_v2_name = f5xc_securemesh_site_v2.example.name
  enhanced_firewall_policies = [
    f5xc_enhanced_firewall_policy.efp1.name,
    f5xc_enhanced_firewall_policy.efp2.name,
  ]
}
