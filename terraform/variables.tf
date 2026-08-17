variable "proxmox_endpoint" {
  type = string
}

variable "proxmox_api_token" {
  type      = string
  sensitive = true
}

variable "ssh_public_key" {
  type = string
}

variable "template_id" {
  type    = number
  default = 9000
}

variable "target_node" {
  type    = string
  default = "homelab"
}
