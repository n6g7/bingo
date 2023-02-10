# Bingo

Bingo automates the creation of DNS records for services served by a reverse proxy.

The original idea was to run alongside [Fabio](https://fabiolb.net/) ([hence](https://fabiolb.net/faq/why-fabio/) [the name](https://www.answers.com/movies-and-television/What_names_is_Nemo_called_by_Dory_in_Finding_Nemo)).

## Architecture

![Bingo Architecture](./arch.png)

## Requirements

- A [supported nameserver](#nameservers)
- A [supported reverse proxy](#reverse-proxies)
- A "service parent domain" to register your service domains under.
  For example, with "svc.local" as a service parent domain, Bingo will:
  - Assume "ownership" of every subdomain of "svc.local" in the nameserver, meaning it will:
    - only ever create subdomains of "svc.local"
    - delete subdomains of "svc.local" it does not recognise
  - For a service named "myapp", create the subdomain "myapp.svc.local"
  - For a service named "home-assistant", create the subdomain "home-assistant.svc.local"
  - etc.

## Usage

### Docker CLI

```bash
docker run -d \
    -e FABIO_HOSTS="host1.local host2.local" \
    -e PIHOLE_URL=http://pihole.local:80 \
    -e PIHOLE_PASSWORD=abc123 \
    -e SERVICE_DOMAIN=svc.local \
    n6g7/bingo
```

### Docker Compose

```yaml
version: "3"

services:
  bingo:
    image: n6g7/bingo
    environment:
      FABIO_HOSTS: host1.local host2.local
      PIHOLE_URL: http://pihole.local:80
      PIHOLE_PASSWORD: "abc123"
      SERVICE_DOMAIN: svc.local
    restart: unless-stopped
```

### Nomad + Consul

```hcl
job "bingo" {
  datacenters = ["dc1"]
  type        = "service"

  group "bingo" {
    network {
      mode = "bridge"
      port "metrics" {}
    }

    service {
      name = "bingo"
      tags = ["metrics"]
      port = "metrics"

      check {
        name     = "http port alive"
        type     = "http"
        path     = "/health"
        interval = "10s"
        timeout  = "2s"
      }
    }

    task "bingo" {
      driver = "docker"

      config {
        image = "n6g7/bingo"
      }

      template {
        destination = "secrets/bingo.env"
        env = true

        data = <<EOH
        FABIO_HOSTS="{{ range service "fabio" }}{{ .Node }}.local {{ end }}"
        PIHOLE_URL=http://pihole.local:80
        PIHOLE_PASSWORD="abc123"
        SERVICE_DOMAIN=svc.local
        PROMETHEUS_LISTEN_ADDR=":{{ env "NOMAD_PORT_metrics" }}"
        EOH
      }

      resources {
        cpu    = 50
        memory = 32
      }
    }
  }
}

```

## Configuration

Bingo aims to require the least configuration possible, however we're not quite there yet.

All configuration is passed as environment variables.

### Minimum config for Fabio and Pi-hole

| Variable name     | Example                   | Description                                              |
| ----------------- | ------------------------- | -------------------------------------------------------- |
| `FABIO_HOSTS`     | `host1.local host2.local` | Hosts where Fabio is running.                            |
| `PIHOLE_URL`      | `http://pihole.local:80`  | Address of the Pi-hole instance.                         |
| `PIHOLE_PASSWORD` | `abc123`                  | Pi-hole admin password.                                  |
| `SERVICE_DOMAIN`  | `svc.local`               | Domain under which service subdomains should be created. |

### Complete config

| Variable name              | Default     | Description                                                                                                                                                                                                              |
| -------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `SERVICE_DOMAIN`           |             | Domain under which service subdomains should be created. Any service with a declared domain that does not match "\*.$SERVICE_DOMAIN" will be ignored. Bingo only ever creates or deletes subdomains of `SERVICE_DOMAIN`. |
| `PROXY_TYPE`               | `fabio`     | The type of proxy to fetch services from. Currently only supports "fabio".                                                                                                                                               |
| `PROXY_POLL_INTERVAL`      | `5s`        | Time interval between requests to reverse proxy.                                                                                                                                                                         |
| `FABIO_HOSTS`              |             | List of space-separated hosts where Fabio is running.                                                                                                                                                                    |
| `FABIO_ADMIN_PORT`         | `9998`      | Fabio's [admin UI port](https://fabiolb.net/ref/ui.addr/).                                                                                                                                                               |
| `FABIO_SCHEME`             | `http`      | URI scheme for Fabio                                                                                                                                                                                                     |
| `TRAEFIK_HOSTS`            |             | List of space-separated hosts where Traefik is running.                                                                                                                                                                  |
| `TRAEFIK_ADMIN_PORT`       | `8080`      | Traefik's [API port](https://doc.traefik.io/traefik/operations/api/).                                                                                                                                                    |
| `TRAEFIK_SCHEME`           | `http`      | URI scheme for Traefik                                                                                                                                                                                                   |
| `TRAEFIK_ENTRYPOINTS`      |             | List of space-separated Traefik entrypoints to watch. Only services mapped to these entry points will be managed.                                                                                                        |
| `NAMESERVER_TYPE`          | `pihole`    | The type of nameserver to managed records in. Supports "pihole" or "route53".                                                                                                                                            |
| `NAMESERVER_POLL_INTERVAL` | `30s`       | Time interval between requests to nameserver.                                                                                                                                                                            |
| `PIHOLE_URL`               |             | Address of the Pi-hole instance.                                                                                                                                                                                         |
| `PIHOLE_PASSWORD`          |             | Pi-hole admin password.                                                                                                                                                                                                  |
| `ROUTE53_HOSTED_ZONE`      |             | Route53 hosted zone name (eg. "sub.domain.com")                                                                                                                                                                          |
| `ROUTE53_TTL`              | `3600`      | TTL of records created in Route53.                                                                                                                                                                                       |
| `AWS_REGION`               | `us-west-1` | The AWS region to connect to when using Route 53. Route 53 is a global service so any region will work, changing the region will only affects latency.                                                                   |
| `AWS_ACCESS_KEY_ID`        |             | When using environment variables to authenticate with AWS, the Access Key ID to use.                                                                                                                                     |
| `AWS_SECRET_ACCESS_KEY`    |             | When using environment variables to authenticate with AWS, the Secret Access Key to use.                                                                                                                                 |
| `AWS_PROFILE`              |             | When using the AWS shared configuration file (usually in `~/.aws/{credentials,config}`) to authenticate with AWS, the name of the profile to use.                                                                        |
| `LOG_LEVEL`                | `INFO`      | Logging verbosity. Supports "TRACE", "DEBUG", "INFO", "WARN", "ERROR" and "FATAL".                                                                                                                                       |
| `MAIN_LOOP_TIMEOUT`        | `1s`        | Lower timeout means faster drift detection at the cost of higher CPU usage.                                                                                                                                              |
| `RECONCILIATION_TIMEOUT`   | `30s`       | Minimum interval between reconciliations.                                                                                                                                                                                |
| `RECONCILER_LOOP_TIMEOUT`  | `1s`        | Lower timeout means faster reconciliation at the cost of higher CPU usage.                                                                                                                                               |
| `PROMETHEUS_LISTEN_ADDR`   | `:9100`     | Address on which the prometheus exporter should listen.                                                                                                                                                                  |
| `PROMETHEUS_METRICS_PATH`  | `/metrics`  | Metrics path for prometheus exporter.                                                                                                                                                                                    |

## Backends

### Reverse proxies

| Name                                  | Status       | Notes                                                                                                        |
| ------------------------------------- | ------------ | ------------------------------------------------------------------------------------------------------------ |
| [Fabio](https://fabiolb.net/)         | ✅ Supported |                                                                                                              |
| [Træfik](https://traefik.io/traefik/) | ✅ Supported | The Træfik backend requires the [Traefik API](https://doc.traefik.io/traefik/operations/api/) to be enabled. |
| [HAProxy]()                           | No issue     |                                                                                                              |

### Nameservers

| Name                                        | Status                                                    | Notes                                                                                                                                                                                      |
| ------------------------------------------- | --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [Pi-hole](https://pi-hole.net/)             | ✅ Supported                                              | Requires the Pi-hole admin password to manage local CNAME records.                                                                                                                         |
| [Route 53](https://aws.amazon.com/route53/) | ✅ Supported                                              | Supports either static credentials (`AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables), shared config file (`AWS_PROFILE`) or IAM role authentication (auto-detected). |
| [pfSense](https://www.pfsense.org/)         | ⏳ [Issue opened](https://github.com/n6g7/bingo/issues/8) |                                                                                                                                                                                            |
