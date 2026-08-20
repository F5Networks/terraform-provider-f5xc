# Example: Origin Pool - Full

data "f5xc_origin_pool" "example" {
  name      = f5xc_origin_pool.example.name
  namespace = f5xc_origin_pool.example.namespace
}
