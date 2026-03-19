output "resource" {
  value = {
    lb = f5xc_http_loadbalancer.lb
    //token = f5xc_token.t1
    //smv2 = f5xc_securemesh_site_v2.smv2
    //domain_name = f5xc_dns_domain.example
    //dns_zone = f5xc_dns_zone.zone
    //origin_pool = f5xc_origin_pool.op
    //healthcheck = f5xc_healthcheck.hc1
    //alert_policy = f5xc_alert_policy.ap1
    //namespace = f5xc_namespace.ns1
  }
}

output "data" {
  value = {
    //lb = data.f5xc_http_loadbalancer.lb
    //smv2 = data.f5xc_securemesh_site_v2.smv2
    //ns = data.f5xc_namespace.ns1
    //op = data.f5xc_origin_pool.op1
    //hc = data.f5xc_healthcheck.hc1
    //ap = data.f5xc_alert_policy.ap1
    //tk = data.f5xc_token.t1
  }
}
