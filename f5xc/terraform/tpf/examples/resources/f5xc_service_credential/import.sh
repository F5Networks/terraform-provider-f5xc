# System-scoped credentials: system/<credential_name>
# vK8s kubeconfig credentials (service_credential_choice.vk8s_kubeconfig): system/<vk8s_namespace>/<credential_name> or <vk8s_namespace>/<credential_name>
# After import, name, expiration_days and service_credential_choice (except vk8s_kubeconfig.vk8s_namespace) are not populated from the API and remain unset.
# Because name and service_credential_choice require replacement, the first plan after import proposes replacing the credential; add lifecycle { ignore_changes = [name, service_credential_choice, expiration_days] } to keep the imported credential.
terraform import f5xc_service_credential.example <namespace>/<credential_name>
