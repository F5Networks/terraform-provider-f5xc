output "resource" {
  value = {
    // lb = f5xc_http_loadbalancer.lb
    // token = f5xc_token.t1
    smv2 = f5xc_securemesh_site_v2.smv2
    // domain_name = f5xc_dns_domain.example
    // dns_zone = f5xc_dns_zone.zone
    // origin_pool = f5xc_origin_pool.op
    // healthcheck = f5xc_healthcheck.hc1
    // alert_policy = f5xc_alert_policy.ap1
    // namespace = f5xc_namespace.ns1
    // vn = f5xc_virtual_network.vn1
    // vs = f5xc_virtual_site.vs1
    // vss = f5xc_voltstack_site.vss1
  }
  sensitive = true
}

/*
output "sensitive" {
  value = {
    ar1 = f5xc_alert_receiver.ar1
  }
  sensitive = true
}
*/

output "data" {
  value = {
    // lb = data.f5xc_http_loadbalancer.lb
    //smv2 = data.f5xc_securemesh_site_v2.smv2
    /*
    ns   = data.f5xc_namespace.ns1
    op   = data.f5xc_origin_pool.op1
    hc   = data.f5xc_healthcheck.hc1
    ap   = data.f5xc_alert_policy.ap1
    tk   = data.f5xc_token.t1
    vk8s = data.f5xc_virtual_k8s.vk8s1
    vn   = data.f5xc_virtual_network.vn1
    vss  = data.f5xc_voltstack_site.vss1
    efp1 = data.f5xc_enhanced_firewall_policy.efp1
    nfw1 = data.f5xc_network_firewall.nfw1
    ar1  = data.f5xc_alert_receiver.ar1
    */
    // sp1 = data.f5xc_service_policy.sp1
    // fpp1 = data.f5xc_forward_proxy_policy.ffp1
    // v1_dns_mon1 = data.f5xc_v1_dns_monitor.v1_dns_mon1
    // smg1 = data.f5xc_site_mesh_group.smg1
    // tcp_lb = data.f5xc_tcp_loadbalancer.tcp-lb1
  }
}
