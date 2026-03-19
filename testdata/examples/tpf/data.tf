/*
data "f5xc_http_loadbalancer" "lb" {
  name = "api-asingla-crapi-lb"
  #tenant = "playground"
  namespace = "default"
}

data "f5xc_securemesh_site_v2" "smv2" {
  name = "coleman-sms-v7"
  #tenant = "playground"
  namespace = "system"
}
*/

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