data "onepassword_item" "skladka" {
  vault = var.onepassword_vault
  title = var.onepassword_item
}

locals {
  onepass_sections = {
    for section in data.onepassword_item.skladka.section:
      section.label => section.field
  }

  db_fields = {
    for field in local.onepass_sections["database"]:
      field.label => field.value
  }

  db_name = local.db_fields["database"]
  db_username = local.db_fields["username"]
  db_password = local.db_fields["password"]
  bucket_backup_hostname = local.db_fields["backup bucket"]
  bucket_backup_accesskey = local.db_fields["backup access key"]
  bucket_backup_secretkey = local.db_fields["backup secret key"]
}
