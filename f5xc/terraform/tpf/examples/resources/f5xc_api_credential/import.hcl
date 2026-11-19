import {
	# System-scoped credentials: system/<credential_name>
	# vK8s KubeConfig credentials (spec.type = ["KUBE_CONFIG"]): system/<vk8s_namespace>/<credential_name> or <vk8s_namespace>/<credential_name>
	# After import, name, expiration_days and spec (except spec.virtual_k8s_namespace) are not populated from the API and remain unset.
	# Because name and spec require replacement, the first plan after import proposes replacing the credential; add lifecycle { ignore_changes = [name, spec, expiration_days] } to keep the imported credential.
	id = "<namespace>/<credential_name>"
	to = f5xc_api_credential.example
}
