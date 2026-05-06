# ---------------------------------------------------------------------------
# Data sources – each block is guarded by its own flag from terraform.tfvars.
# Set the corresponding *_data variable to true to activate a data source.
# ---------------------------------------------------------------------------

data "f5xc_namespace" "ns1" {
  count      = var.f5xc_namespace_data ? 1 : 0
  depends_on = [f5xc_namespace.ns1]
  name       = f5xc_namespace.ns1[0].name
}

data "f5xc_http_loadbalancer" "lb" {
  count      = var.f5xc_http_loadbalancer_data ? 1 : 0
  depends_on = [f5xc_http_loadbalancer.lb]
  name       = "ck-lb-test-01"
  namespace  = "default"
}

data "f5xc_securemesh_site_v2" "smv2" {
  count      = var.f5xc_securemesh_site_v2_data ? 1 : 0
  name       = "ck-smv2-test-01"
  namespace  = "system"
}

data "f5xc_origin_pool" "op1" {
  count      = var.f5xc_origin_pool_data ? 1 : 0
  depends_on = [f5xc_origin_pool.op]
  name       = "test-op-123"
  namespace  = "default"
}

data "f5xc_healthcheck" "hc1" {
  count      = var.f5xc_healthcheck_data ? 1 : 0
  depends_on = [f5xc_healthcheck.hc1]
  name       = "test-hc-1"
  namespace  = "default"
}

data "f5xc_alert_policy" "ap1" {
  count      = var.f5xc_alert_policy_data ? 1 : 0
  depends_on = [f5xc_alert_policy.ap1]
  name       = "test-alert-policy-1"
  namespace  = "default"
}

data "f5xc_token" "t1" {
  count      = var.f5xc_token_data ? 1 : 0
  depends_on = [f5xc_token.t1]
  name       = "ck-test-token-01"
  namespace  = "system"
}

data "f5xc_virtual_k8s" "vk8s1" {
  count      = var.f5xc_virtual_k8s_data ? 1 : 0
  depends_on = [f5xc_virtual_k8s.vk8s1]
  name       = "test-vk8s-1"
  namespace  = "ck-test-ns1"
}

data "f5xc_virtual_network" "vn1" {
  count      = var.f5xc_virtual_network_data ? 1 : 0
  depends_on = [f5xc_virtual_network.vn1]
  name       = "test-vn-1"
  namespace  = "system"
}

data "f5xc_voltstack_site" "vss1" {
  count      = var.f5xc_voltstack_site_data ? 1 : 0
  depends_on = [f5xc_voltstack_site.vss1]
  name       = "test-site-1"
  namespace  = "system"
}

data "f5xc_enhanced_firewall_policy" "efp1" {
  count      = var.f5xc_enhanced_firewall_policy_data ? 1 : 0
  depends_on = [f5xc_enhanced_firewall_policy.efp1]
  name       = "ck-test-efp1"
  namespace  = "system"
}

data "f5xc_network_firewall" "nfw1" {
  count      = var.f5xc_network_firewall_data ? 1 : 0
  depends_on = [f5xc_network_firewall.nfw1]
  name       = "ck-test-nfw-1"
  namespace  = "system"
}

data "f5xc_network_policy_rule" "npr1" {
  count      = var.f5xc_network_policy_rule_data ? 1 : 0
  depends_on = [f5xc_network_policy_rule.nfwr1]
  name       = "ck-test-nfwr1"
  namespace  = "system"
}

data "f5xc_alert_receiver" "ar1" {
  count      = var.f5xc_alert_receiver_data ? 1 : 0
  depends_on = [f5xc_alert_receiver.ar1]
  name       = "ck-test-ar-1"
  namespace  = "system"
}

data "f5xc_service_policy" "sp1" {
  count      = var.f5xc_service_policy_data ? 1 : 0
  depends_on = [f5xc_service_policy.sp1]
  name       = "ck-sp-01"
  namespace  = "default"
}

data "f5xc_forward_proxy_policy" "fpp1" {
  count      = var.f5xc_forward_proxy_policy_data ? 1 : 0
  depends_on = [f5xc_forward_proxy_policy.fpp1]
  name       = "ck-fpp-01"
  namespace  = "system"
}

data "f5xc_v1_dns_monitor" "v1_dns_mon1" {
  count      = var.f5xc_v1_dns_monitor_data ? 1 : 0
  depends_on = [f5xc_v1_dns_monitor.v1_dns_mon1]
  name       = "ck-test-dns-mon-01"
  namespace  = "system"
}

data "f5xc_site_mesh_group" "smg1" {
  count      = var.f5xc_site_mesh_group_data ? 1 : 0
  depends_on = [f5xc_site_mesh_group.smg1]
  name       = "ck-smg-01"
  namespace  = "system"
}

data "f5xc_tcp_loadbalancer" "tcp-lb1" {
  count     = var.f5xc_tcp_loadbalancer_data ? 1 : 0
  name      = "ck-tcp-lb-1"
  namespace = "default"
}
