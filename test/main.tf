variable "secret_one" {
  type = string
}

variable "secret_two" {
  type = string
}

locals {
  something_else = {
    s1 = var.secret_one
    s2 = var.secret_two
  }
}
