variable "database_url" {
  type    = string
  default = "postgres://app_user:app_password@localhost:15432/app_db?sslmode=disable"
}

variable "dev_url" {
  type    = string
  default = "docker://postgres/16/dev?search_path=public"
}

env "local" {
  url = var.database_url
  dev = var.dev_url

  schema {
    src = "file://assets/db/schema.sql"
  }

  migration {
    dir = "file://assets/migrations?format=golang-migrate"
  }
}
