resource "f5xc_app_firewall" "example" {
  name      = "example_app_firewall"
  namespace = "example_namespace"
  enhance_with_ai_choice = {
    disable_ai_enhancements = true
  }
  enforcement_mode_choice = {
    blocking = true
  }
  detection_setting_choice = {
    default_detection_settings = true
  }
  allowed_response_codes_choice = {
    allowed_response_codes = {
      allowed_response_codes = "example_allowed_response_codes"
    }
  }
  anonymization_setting = {
    default_anonymization = true
  }
  blocking_page_choice = {
    use_default_blocking_page = true
  }
}
