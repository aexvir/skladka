resource "kubernetes_deployment" "skladka" {
  metadata {
    name      = "skladka"
    namespace = kubernetes_namespace.skladka.id
    labels = {
      app = "skladka"
    }
  }

  spec {
    replicas = var.replicas

    selector {
      match_labels = {
        app = "skladka"
      }
    }

    template {
      metadata {
        labels = {
          app = "skladka"
        }
      }

      spec {
        container {
          image = "alexviscreanu/skladka:${var.image_tag}"
          name  = "skladka"

          port {
            container_port = 3000
          }

          env_from {
            secret_ref {
              name = kubernetes_secret.skladka.metadata[0].name
            }
          }

          resources {
            limits = {
              cpu    = "500m"
              memory = "512Mi"
            }
            requests = {
              cpu    = "250m"
              memory = "256Mi"
            }
          }

          liveness_probe {
            http_get {
              path = "/health"
              port = 3000
            }
            initial_delay_seconds = 10
            period_seconds        = 30
          }
        }
      }
    }
  }

  depends_on = [
    kubernetes_manifest.db
  ]
}

resource "kubernetes_service" "skladka" {
  metadata {
    name      = "skladka"
    namespace = kubernetes_namespace.skladka.id
  }

  spec {
    selector = {
      app = kubernetes_deployment.skladka.metadata[0].labels.app
    }

    port {
      port        = 80
      target_port = 3000
    }

    type = var.service_type
  }
}

resource "kubernetes_secret" "skladka" {
  metadata {
    name      = "skladka"
    namespace = kubernetes_namespace.skladka.id
  }

  data = {
    SKD_POSTGRES_HOST = "skladka-db-rw"
    SKD_POSTGRES_USER = local.db_username
    SKD_POSTGRES_PASS = local.db_password
    SKD_POSTGRES_DB   = local.db_name

    SKD_ENCRYPTION_KEY  = local.encryption_key
    SKD_ENCRYPTION_SALT = local.encryption_salt
  }

  depends_on = [
    kubernetes_manifest.db
  ]
}

resource "kubernetes_ingress_v1" "skladka" {
  metadata {
    name      = "skladka"
    namespace = kubernetes_namespace.skladka.id
    annotations = {
      "cert-manager.io/cluster-issuer" : "letsencrypt"
    }
  }
  spec {
    rule {
      host = "paste.xvr.sh"
      http {
        path {
          path = "/"
          backend {
            service {
              name = kubernetes_service.skladka.metadata[0].name
              port {
                number = 80
              }
            }
          }
        }
      }
    }
    tls {
      hosts = [
        "paste.xvr.sh"
      ]
      secret_name = "skladka-cert"
    }
  }
}
