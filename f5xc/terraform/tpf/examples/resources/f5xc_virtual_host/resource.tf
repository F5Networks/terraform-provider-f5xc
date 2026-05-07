resource "f5xc_virtual_host" "example" {
  name      = "example_virtual_host"
  namespace = "example_namespace"
  challenge_type_choice = {
    captcha_challenge = {
      captcha_challenge = "example_captcha_challenge"
    }
  }
  trust_client_ip_headers_choice = {
    disable_trust_client_ip_headers = true
  }
}
