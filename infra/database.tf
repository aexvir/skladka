resource "kubernetes_manifest" "db" {
  computed_fields = ["spec.postgresql"]

  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "Cluster"

    metadata = {
      name      = "skladka-db"
      namespace = kubernetes_namespace.skladka.id
    }

    spec = {
      instances = 1

      # PostgreSQL configuration
      postgresql = {
        parameters = {
          "max_connections"      = "100"
          "shared_buffers"       = "128MB"
          "work_mem"             = "4MB"
          "maintenance_work_mem" = "64MB"
        }
      }

      # Resource requirements
      resources = {
        requests = {
          memory = "512Mi"
          cpu    = "200m"
        }
        limits = {
          memory = "512Mi"
          cpu    = "500m"
        }
      }

      # Storage configuration
      storage = {
        size         = "5Gi"
        storageClass = "longhorn"
      }

      # Backup configuration
      backup = {
        retentionPolicy = "30d"
        barmanObjectStore = {
          destinationPath = "skladka"
          endpointURL     = local.bucket_backup_hostname
          s3Credentials = {
            accessKeyId = {
              name = kubernetes_secret.backup_credentials.metadata[0].name
              key  = "ACCESS_KEY_ID"
            }
            secretAccessKey = {
              name = kubernetes_secret.backup_credentials.metadata[0].name
              key  = "SECRET_ACCESS_KEY"
            }
          }
        }
      }

      # Bootstrap configuration
      bootstrap = {
        initdb = {
          database = "skladka"
          owner    = "popelar"
          secret = {
            name = kubernetes_secret.db_credentials.metadata[0].name
          }
        }
      }

      managed = {
        services = {
          disabledDefaultServices = ["ro", "r"]
        }
      }
    }
  }
}

resource "kubernetes_secret" "db_credentials" {
  metadata {
    name      = "skladka-db-credentials"
    namespace = kubernetes_namespace.skladka.id
  }

  data = {
    username = local.db_username
    password = local.db_password
  }
}

resource "kubernetes_secret" "backup_credentials" {
  metadata {
    name      = "skladka-backup-credentials"
    namespace = kubernetes_namespace.skladka.id
  }

  data = {
    ACCESS_KEY_ID     = local.bucket_backup_accesskey
    SECRET_ACCESS_KEY = local.bucket_backup_secretkey
  }
}
