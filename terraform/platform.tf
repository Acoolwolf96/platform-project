locals {
  teams = {
    "team-checkout" = { cpu_limit = "1", memory_limit = "1Gi" }
    "team-search"   = { cpu_limit = "1", memory_limit = "1Gi" }
    "team-billing"  = { cpu_limit = "500m", memory_limit = "512Mi" }
  }
}

resource "kubernetes_namespace" "team" {
  for_each = local.teams
  metadata {
    name = each.key
    labels = {
      "managed-by" = "terraform"
      "team"       = each.key
    }
  }
}

resource "kubernetes_resource_quota" "team" {
  for_each = local.teams
  metadata {
    name      = "${each.key}-quota"
    namespace = kubernetes_namespace.team[each.key].metadata[0].name
  }
  spec {
    hard = {
      "requests.cpu"    = each.value.cpu_limit
      "requests.memory" = each.value.memory_limit
      "limits.cpu"      = each.value.cpu_limit
      "limits.memory"   = each.value.memory_limit
      "pods"            = "10"
    }
  }
}

resource "kubernetes_network_policy" "deny_cross_namespace" {
  for_each = local.teams
  metadata {
    name      = "deny-cross-namespace"
    namespace = kubernetes_namespace.team[each.key].metadata[0].name
  }
  spec {
    pod_selector {}
    policy_types = ["Ingress"]
    ingress {
      from {
        namespace_selector {
          match_labels = {
            "team" = each.key
          }
        }
      }
    }
  }
}
