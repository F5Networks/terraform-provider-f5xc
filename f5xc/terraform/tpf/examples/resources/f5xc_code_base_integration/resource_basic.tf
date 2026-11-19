# Example: Code Base Integration - Basic

resource "f5xc_code_base_integration" "example" {
  name        = "example-code-base-integration"
  namespace   = "default"
  description = "Code base integration example"

  code_base_integration = {
    type_choice = {
      gitlab = {
        access_token = {
          secret_info_choice = {
            clear_secret_info = {
              url = "string:///dGVzdC10b2tlbg=="
            }
          }
        }
        url        = "https://gitlab.example.com/org/repo"
        username   = "testuser"
        verify_ssl = false
      }
    }
  }
}
