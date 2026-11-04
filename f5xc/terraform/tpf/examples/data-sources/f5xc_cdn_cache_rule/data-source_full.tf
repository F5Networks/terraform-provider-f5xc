# Example: Cdn Cache Rule - Full

data "f5xc_cdn_cache_rule" "example" {
  name      = f5xc_cdn_cache_rule.example.name
  namespace = f5xc_cdn_cache_rule.example.namespace
}
