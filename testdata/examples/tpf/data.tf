/*
data "f5xc_http_loadbalancer" "lb" {
  depends_on = [f5xc_http_loadbalancer.lb]
  //name = "ck-lb-test-01"
  name = "api-asingla-crapi-lb"
  namespace = "default"
}
*/

/*
data "f5xc_securemesh_site_v2" "smv2" {
  depends_on = [f5xc_securemesh_site_v2.smv2]
  name = "ck-smv2-test-01"
  namespace = "system"
}
*/

/*
data "f5xc_namespace" "ns1" {
  depends_on = [f5xc_namespace.ns1]
  name       = "ck-test-ns1"
}

data "f5xc_origin_pool" "op1" {
  depends_on = [f5xc_origin_pool.op]
  name       = "test-op-123"
  namespace  = "default"
}

data "f5xc_healthcheck" "hc1" {
  depends_on = [f5xc_healthcheck.hc1]
  name       = "test-hc-1"
  namespace  = "default"
}

data "f5xc_alert_policy" "ap1" {
  depends_on = [f5xc_alert_policy.ap1]
  name       = "test-alert-policy-1"
  namespace  = "default"
}

data "f5xc_token" "t1" {
  depends_on = [f5xc_token.t1]
  name       = "ck-test-token-01"
  namespace  = "system"
}

data "f5xc_virtual_k8s" "vk8s1" {
  depends_on = [f5xc_virtual_k8s.vk8s1]
  name       = "test-vk8s-1"
  namespace  = data.f5xc_namespace.ns1.name
}

data "f5xc_virtual_network" "vn1" {
  depends_on = [f5xc_virtual_network.vn1]
  name      = "test-vn-1"
  namespace  = "system"
}

data "f5xc_voltstack_site" "vss1" {
    depends_on = [f5xc_voltstack_site.vss1]
    name       = "test-site-1"
    namespace  = "system"
}

data "f5xc_enhanced_firewall_policy" "efp1" {
    depends_on = [f5xc_enhanced_firewall_policy.efp1]
    name       = "ck-test-efp1"
    namespace  = "system"
}

data "f5xc_network_firewall" "nfw1" {
    depends_on = [f5xc_network_firewall.nfw1]
    name       = "ck-test-nfw-1"
    namespace  = "system"
}

data "f5xc_network_policy_rule" "npr1" {
    depends_on = [f5xc_network_policy_rule.nfwr1]
    name       = "ck-test-nfwr1"
    namespace  = "system"
}

data "f5xc_alert_receiver" "ar1" {
  depends_on = [f5xc_alert_receiver.ar1]
  name       = "ck-test-ar-1"
  namespace  = "system"
}
*/

/*
data "f5xc_service_policy" "sp1" {
  depends_on = [f5xc_service_policy.sp1]
  name = "ck-sp-01"
  namespace = "system"
}

data "f5xc_forward_proxy_policy" "ffp1" {
  depends_on = [f5xc_forward_proxy_policy.fpp1]
  name = "ck-fpp-01"
  namespace = "system"
}
*/

/*
data "f5xc_v1_dns_monitor" "v1_dns_mon1" {
  depends_on = [f5xc_v1_dns_monitor.v1_dns_mon1]
  name = "ck-test-dns-mon-01"
  namespace = "system"
}
*/
/*
data "f5xc_site_mesh_group" "smg1" {
  depends_on = [f5xc_site_mesh_group.smg1]
  name = "ck-smg-01"
  namespace = "system"
}

data "f5xc_tcp_loadbalancer" "tcp-lb1" {
  depends_on = [f5xc_tcp_loadbalancer.tcp-lb1]
  name = "ck-tcp-lb-1"
  namespace = "default"
}
*/