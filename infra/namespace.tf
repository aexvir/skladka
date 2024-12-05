resource "kubernetes_namespace" "skladka" {
  metadata {
    name = var.namespace
  }
}
