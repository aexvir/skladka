variable "onepassword_token" {
  description = "token to authenticate against the onepassword cli"
  type        = string
  default     = "token"
}

variable "onepassword_vault" {
  description = "id of the vault where the secrets are stored"
  type        = string
  default     = "vault"
}

variable "onepassword_item" {
  description = "if of the item inside the vault where the secrets are stored"
  type        = string
  default     = "item"
}

variable "namespace" {
  type    = string
  default = "skladka"
}

variable "replicas" {
  description = "Number of replicas for the skladka deployment"
  type        = number
  default     = 2
}

variable "image_tag" {
  description = "Tag of the skladka image to deploy"
  type        = string
  default     = "latest"
}

variable "service_type" {
  description = "Type of Kubernetes service to create"
  type        = string
  default     = "ClusterIP"
}
