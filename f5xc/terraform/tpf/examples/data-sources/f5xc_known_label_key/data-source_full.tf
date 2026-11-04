# Example: Known Label Key - Full

data "f5xc_known_label_key" "example" {
  key       = f5xc_known_label_key.example.key
  namespace = f5xc_known_label_key.example.namespace
}
