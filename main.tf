terraform {
  required_providers {
    apko = {
      source  = "chainguard-dev/apko"
      version = ">= 0.29.11"
    }
    oci = {
      source  = "chainguard-dev/oci"
      version = ">= 0.0.25"
    }
    ko = {
      source  = "chainguard-dev/ko"
      version = ">= 0.0.4"
    }
  }
}

locals {
  uid = 10000
  gid = 10000
  dir_list = [
    "/shared",
    "/shared/html",
    "/shared/tftpboot",
    "/shared/log",
    "/conf",
    "/conf/ironic",
    "/data",
  ]

  dir_parts = [for d in local.dir_list : split("/", d) if d != ""]
  dir_paths = sort(distinct(flatten([for p in local.dir_parts : [for d in p : join("/", slice(p, 0, index(p, d) + 1))]])))
  paths = concat([
    for p in local.dir_paths : {
      path        = p
      type        = "directory"
      permissions = 511
      recursive   = length(compact(split("/", p))) == 1
      source      = ""
      uid         = local.uid
      gid         = local.gid
    }
    ], [{
      path        = "/etc/ironic"
      type        = "directory"
      permissions = 511
      recursive   = false
      source      = ""
      uid         = local.uid
      gid         = local.gid
  }])
}

resource "apko_build" "default" {
  config = {
    accounts = {
      run-as = "ironic"
      users = [{
        uid      = local.uid
        gid      = local.gid
        homedir  = "/var/lib/ironic"
        shell    = "/usr/bin/python3"
        username = "ironic"
      }]
      groups = [{
        gid       = local.gid
        groupname = "ironic"
        members   = ["ironic"]
      }]
    }
    entrypoint = {
      command        = "/usr/bin/python3.13"
      type           = ""
      shell-fragment = ""
      services       = {}
    }
    annotations = {
      "org.opencontainers.image.title"       = "Ironic Image"
      "org.opencontainers.image.description" = "Ironic Image"
      "org.opencontainers.image.version"     = "1.0.0"
    }
    contents = {
      repositories = [
        "https://packages.wolfi.dev/os",
        "https://metal3-community.github.io/ironic-packages/wolfi"
      ]
      keyring = [
        "https://packages.wolfi.dev/os/wolfi-signing.rsa.pub",
        "https://metal3-community.github.io/ironic-packages/melange.rsa.pub"
      ]
      packages = [
        "ca-certificates-bundle",
        "wolfi-baselayout",
        "python-3.13",
        "py3.13-pbr",
        "py3.13-alembic",
        "py3.13-bcrypt",
        "py3.13-cheroot",
        "py3.13-jinja2",
        "py3.13-jsonpatch",
        "py3.13-wrapt",
        "py3.13-prettytable",
        "py3.13-websockify",
        "py3-proliantutils",
        "py3-zeroconf",
        "py3-ironic-prometheus-exporter",
        "py3-ironic",
        "ipmitool",
      ]
      runtime_repositories = []
      build_repositories   = []
    }

    archs       = ["x86_64", "aarch64"]
    cmd         = ""
    environment = {}
    include     = ""
    layering    = { budget = 0, strategy = "" }
    paths       = local.paths
    stop-signal = "SIGTERM"
    vcs-url     = "https://github.com/metal3-community/ironic-image"
    volumes     = []
    work-dir    = "/var/lib/ironic"
  }
  repo = "ghcr.io/metal3-community/ironic-image"
}

resource "ko_image" "default" {
  base_image = apko_build.default.id
  importpath = "github.com/metal3-community/metal-boot/cmd/metal-boot"
  repo       = "ghcr.io/metal3-community/metal-boot"
  platforms  = ["linux/amd64", "linux/arm64"]
}

output "ipa_image_ref" {
  value = ko_image.default.id
}
