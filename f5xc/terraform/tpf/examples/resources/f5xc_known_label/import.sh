# Composite-key import ID format: <namespace>/<key>=<value> (for example: shared/env=prod or shared/ves.io/region=ves-io-par).
# For labels with an empty value, keep the trailing '=': <namespace>/<key>=
terraform import f5xc_known_label.example <namespace>/<key>=<value>
