# Example: Api Testing - Full

resource "f5xc_api_testing" "example" {
  name                = "example-api-testing"
  namespace           = "default"
  description         = "API testing example"
  custom_header_value = "my-test-header-value"

  labels = {
    env  = "example"
    tier = "api"
  }

  domains = [
    {
      domain                    = "api.example.com"
      allow_destructive_methods = true
      credentials = [
        {
          credential_name = "cred-admin-basic"
          credentials_choice = {
            basic_auth = {
              user = "adminuser"
              password = {
                secret_info_choice = {
                  clear_secret_info = {
                    url = "string:///YWRtaW5wYXNzd29yZA=="
                  }
                }
              }
            }
          }
          role_choice = {
            admin = true
          }
        },
        {
          credential_name = "cred-std-apikey"
          credentials_choice = {
            api_key = {
              key = "Authorization"
              value = {
                secret_info_choice = {
                  clear_secret_info = {
                    url = "string:///c3RkYXBpa2V5"
                  }
                }
              }
            }
          }
          role_choice = {
            standard = true
          }
        }
      ]
    },
    {
      domain = "staging.example.com"
      credentials = [
        {
          credential_name = "cred-bearer"
          credentials_choice = {
            bearer_token = {
              token = {
                secret_info_choice = {
                  clear_secret_info = {
                    url = "string:///c3RhZ2luZ3Rva2Vu"
                  }
                }
              }
            }
          }
          role_choice = {
            standard = true
          }
        }
      ]
    }
  ]

  frequency_choice = {
    every_day = true
  }
}
