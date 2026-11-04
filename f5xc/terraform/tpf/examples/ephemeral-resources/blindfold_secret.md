---
page_title: "f5xc_blindfold_secret Ephemeral Resource - F5 XC Distributed Cloud"
subcategory: "Identity & Secret Management"
description: |-
  Client-side Blindfold envelope encryption for F5 Distributed Cloud secrets.
  Wraps a plaintext payload with the tenant public RSA key + AES-256-GCM DEK
  and binds the ciphertext to a Secret Policy, so raw secrets never touch
  Terraform state or the F5XC control-plane database.
---

# f5xc_blindfold_secret (Ephemeral Resource)

`ephemeral.f5xc_blindfold_secret` performs client-side envelope encryption
against the F5 Distributed Cloud (F5XC) Secret Management APIs. It:

1. Fetches the tenant public RSA key from
   `GET /secret_management/get_public_key`.
2. Fetches the bound Secret Policy document from
   `GET /secret_management/namespaces/{namespace}/secret_policys/{policy_name}/get_policy_document`.
3. Generates a single-use 256-bit AES-GCM data-encryption key (DEK) and a
   12-byte GCM nonce.
4. Encrypts the plaintext payload with AES-256-GCM.
5. Wraps the DEK using the blinded-RSA scheme implemented by
   `gopkg.volterra.us/stdlib/blindfoldutil.EncryptSecret`
   (see [Envelope Format](#envelope-format) below).
6. Emits the packed envelope as a `string:///<base64>` URL suitable for direct
   consumption by any F5XC resource field that accepts
   `blindfold_secret_info.location`.

Because this is an **ephemeral resource** (Terraform 1.10+), the raw `content`
value exists only in memory during `terraform apply` and is **never written to
`.tfstate`, plan files, or the graph diff**. Only the `location`, `policy_id`,
`key_version`, and `tenant` attributes are exported for downstream consumption.

The `location` output is designed to be assigned directly (no `string://`
concatenation needed) to any resource attribute typed `WriteOnly`, e.g.
`f5xc_http_loadbalancer` `blindfold_secret_info.location`. See
[guides/blindfold_secret](../guides/blindfold_secret.md) for the end-to-end
architecture.

## Requirements

- Terraform `>= 1.10.0` (ephemeral resources).
- F5XC API credentials configured on the provider block.
- A Secret Policy already provisioned in F5XC (see `f5xc_secret_policy`).

## Example Usage

```terraform
terraform {
  required_version = ">= 1.10.0"
  required_providers {
    f5xc = {
      source = "f5networks/f5xc"
    }
  }
}

# Ephemeral local_file keeps the raw TLS key out of tfstate.
ephemeral "local_file" "tls_key" {
  filename = "${path.module}/tls.key"
}

# Native Blindfold envelope encryption. Plaintext lives only in memory.
ephemeral "f5xc_blindfold_secret" "tls_key_blindfolded" {
  namespace   = "shared"
  policy_name = "production-secret-policy"
  content     = ephemeral.local_file.tls_key.content
}

# Only the non-sensitive string:/// ciphertext URL is persisted to tfstate.
resource "f5xc_http_loadbalancer" "example" {
  name      = "web-lb"
  namespace = "production"

  # ... other required attributes elided for brevity ...

  # tls_parameters.tls_certificates[*].private_key.blindfold_secret_info.location
  # accepts the ciphertext URL produced above.
  # tls_parameters = {
  #   tls_certificates = [{
  #     private_key = {
  #       blindfold_secret_info = {
  #         location = ephemeral.f5xc_blindfold_secret.tls_key_blindfolded.location
  #       }
  #     }
  #   }]
  # }
}
```

### Rotating an API token

```terraform
ephemeral "f5xc_blindfold_secret" "api_token" {
  namespace   = "shared"
  policy_name = "third-party-integration-policy"
  content     = var.upstream_api_token # marked sensitive in variables.tf
}

resource "f5xc_alert_receiver" "slack" {
  # ...
  slack {
    webhook = {
      blindfold_secret_info = {
        location = ephemeral.f5xc_blindfold_secret.api_token.location
      }
    }
  }
}
```

## Schema

### Required

- `namespace` (String) — Namespace of the F5XC Secret Policy that authorizes
  runtime decryption of the resulting ciphertext.
- `policy_name` (String) — Name of the F5XC Secret Policy that the ciphertext
  must be bound to.
- `content` (String, Sensitive) — Plaintext secret payload to blindfold. Held
  only in memory during `terraform apply` — never persisted to `.tfstate`,
  plan files, or debug logs.

### Optional

- `key_version` (Number) — Public-key version to use. Defaults to the latest
  version returned by the tenant KMS (`GET /public/get_public_key`).

### Read-Only

- `location` (String, Sensitive) — The blindfolded ciphertext in
  `string:///<base64(envelope)>` form. Feed this into any F5XC resource field
  that accepts `blindfold_secret_info.location`.
- `policy_id` (Number) — Numeric policy identifier bound into the ciphertext
  header. Useful for observability and audit.
- `tenant` (String) — Tenant returned by the Secret Policy document.

## Envelope Format

The `location` value has the form `string:///<base64(bundle)>`, where `bundle`
is the exact byte layout produced by
`gopkg.volterra.us/stdlib/blindfoldutil.EncryptSecret` and consumed by wingman:

```
[ uint16 version = 0 ]
[ uint16 tenant_len ][ tenant_bytes ]
[ uint32 key_version ]
[ uint64 policy_id ]
[ uint8  algo_id = 2  (AES-256-GCM) ]
[ uint32 exp_len ][ exp_bytes ]
[ uint32 mod_len ][ mod_bytes ]
[ uint32 enc_key_len ][ enc_key_bytes ]
[ enc_msg_bytes ]
```

- `mod_bytes`, `exp_bytes` — the tenant RSA public key, as returned by
  `GET /secret_management/get_public_key`.
- `enc_key_bytes = M^(e·τ) mod N`, with
  `M = 0xdeadbeef || gcm_nonce(12) || aes_key(32) || random_pad(mod_len − 50)`
  and `τ = 2^(k−1) + 2·policy_id + 1` (k = 32). This is the blinded-RSA
  wrapping used by F5XC — not standard RSA-OAEP.
- `enc_msg_bytes = AES-256-GCM.Seal(aes_key, gcm_nonce, plaintext, aad=nil)`
  (ciphertext concatenated with the 16-byte GCM tag).
- `tenant_bytes` **must equal the tenant that owns the RSA public key**
  (`data.tenant` from `get_public_key`). Using the Secret Policy's tenant
  (e.g. `ves-io` for shared policies) causes wingman to look up the wrong
  private key and return
  `Invalid blindfold secret - make sure encrypted secret is secured using blindfold`.

The DEK is scrubbed from process memory immediately after wrapping.

## Zero-State Guarantee

Ephemeral resources are opened once per apply and never written to disk.
Assertions:

- `content` — never appears in `.tfstate`, plan JSON, or provider debug logs
  (marked `Sensitive`).
- `location` — persisted only if a downstream **regular** resource references
  it into its own state.
- Renaming `namespace` or `policy_name` re-opens the resource; there is no
  "update" concept for ephemeral resources.

## Security Model

- Confidentiality relies on standard cryptography (AES-256-GCM for payload
  encryption, plus the F5XC blinded-RSA DEK-wrapping construction
  `C = M^(e·τ) mod N` described in [Envelope Format](#envelope-format)) and on
  the confidentiality of the tenant KMS private key — not on the secrecy of
  the encryption code (Kerckhoffs's Principle).
- Runtime decryption occurs only in the volatile RAM of authorized Customer
  Edge (CE) nodes after the F5XC control plane validates the bound Secret
  Policy.
- No `vesctl` binary or external subprocess is required, so this resource is
  fully compatible with HCP Terraform, Spacelift, env0, and other managed
  Terraform runners.

## Import

Ephemeral resources cannot be imported: they have no persistent state to
adopt.
