# Example: Known Label - Basic

resource "f5xc_known_label_key" "key" {
  key       = "example-kl"
  namespace = "shared"
}

resource "f5xc_known_label" "example" {
  key       = "example-kl"
  value     = "example-val"
  namespace = "shared"

  depends_on = [f5xc_known_label_key.key]
}
