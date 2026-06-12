variable "KESTREL_TAGS" {
  default = "ghcr.io/captdany/kestrel:latest"
}
variable "SCRAPER_TAGS" {
  default = "ghcr.io/captdany/kestrel-scraper:latest"
}

target "_common" {
  context = "."
  dockerfile = "Dockerfile"
  provenance = false
  cache-from = ["type=gha"]
  cache-to = ["type=gha,mode=max"]
}

target "kestrel" {
  inherits = ["_common"]
  target = "kestrel"
  tags = split(",", KESTREL_TAGS)
  labels = {
    "org.opencontainers.image.title" = "kestrel"
    "org.opencontainers.image.description" = "Self-hosted purchase planner"
    "org.opencontainers.image.url" = "https://github.com/CaptDany/kestrel"
    "org.opencontainers.image.source" = "https://github.com/CaptDany/kestrel"
    "org.opencontainers.image.licenses" = "MIT"
  }
}

target "scraper" {
  inherits = ["_common"]
  target = "scraper"
  tags = split(",", SCRAPER_TAGS)
  labels = {
    "org.opencontainers.image.title" = "kestrel-scraper"
    "org.opencontainers.image.description" = "Playwright scraper sidecar for kestrel"
    "org.opencontainers.image.url" = "https://github.com/CaptDany/kestrel"
    "org.opencontainers.image.source" = "https://github.com/CaptDany/kestrel"
    "org.opencontainers.image.licenses" = "MIT"
  }
}
