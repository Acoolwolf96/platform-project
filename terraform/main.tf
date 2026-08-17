locals {
  k3s_nodes = {
    "k3s-server" = { vmid = 210, cores = 2, memory = 3072, ip = "10.0.0.210/24" }
    "k3s-agent"  = { vmid = 211, cores = 1, memory = 1536, ip = "10.0.0.211/24" }
  }
}

resource "proxmox_virtual_environment_vm" "k3s_node" {
  for_each  = local.k3s_nodes
  name      = each.key
  node_name = var.target_node
  vm_id     = each.value.vmid

  clone {
    vm_id = var.template_id
    full  = true
  }

  cpu    { cores = each.value.cores }
  memory { dedicated = each.value.memory }

  disk {
    datastore_id = "local-lvm"
    interface    = "scsi0"
    size         = 10
  }

  initialization {
    ip_config {
      ipv4 {
        address = each.value.ip
        gateway = "10.0.0.1"
      }
    }
    user_account {
      username = "acoolwolf"
      keys     = [var.ssh_public_key]
    }
  }

  agent { enabled = true }
}

output "k3s_node_ips" {
  value = { for k, v in local.k3s_nodes : k => v.ip }
}
