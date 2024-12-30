# skladka

{{< figure src="images/logo-color.svg" width="300px" >}}

minimalistic pastebin service written in go, deployable as single binary

## codebase

this is a service where go is not only used for the backend but embraced as much as possible for the whole stack

there's no javascript build system for the frontend, no bash scripts for automation, and nothing runs in containers

the goal of this project is to create a proof of concept for a web service using tools I didn't use before (mainly templ, sqlc, otel) and develop a base for future projects

### building blocks

- [go-chi/chi](https://github.com/go-chi/chi) for routing
- [ardanlabs/conf](https://github.com/ardanlabs/conf) for managing configuration
- [opentelemetry](https://github.com/open-telemetry/opentelemetry-go) for the observability stack
- [grafana](https://github.com/grafana) for visualizing metrics, traces and logs
- [sqlc-dev/sqlc](https://github.com/sqlc-dev/sqlc) for generating type-safe sql code
- [a-h/templ](https://github.com/a-h/templ) for generating type-safe html code
- [opentofu](https://github.com/opentofu) for managing infra and deployments

## development

[mage](https://github.com/magefile/mage) is used as entrypoint for all development tasks, so make sure to have it installed

you can install it via go install by running
```shell
go install github.com/magefile/mage@latest
```
then just run `mage` in the root of this reposiroty in order to discover all tasks

### services

[process-compose](https://github.com/F1bonacc1/process-compose) is used to orchestrate services locally during development

you can install it via homebrew by running
```shell
brew install f1bonacc1/tap/process-compose
```

then it's enough to run `mage dev` to spin up the project locally alongside all its dependencies and observability stack

{{< callout type="warning" >}}
note: binaries like postgres, grafana, mimir, loki, tempo and alloy are not downloaded automagically yet, so you'll need them installed on your machine

nix will solve this 😄
{{< /callout >}}

for a more minimal setup, you can just run `process-compose up postgres` to start the database and then spin up the service through your preferred ide

or `process-compose up postgres skladka` to start both the database and the service with hot reload using [air](https://github.com/air-verse/air) if it's installed

if all binaries are available and process compose starts correctly, the following services should be available

{{< cards cols="1" >}}
  {{< card link="http://localhost:3000" title="skladka service" icon="trash" tag="localhost:3000" >}}
  {{< card link="http://localhost:3000" title="database" icon="database" tag="localhost:2345" >}}
  {{< card link="http://localhost:4500" title="documentation" icon="document" tag="localhost:4500" >}}
  {{< card link="http://localhost:4000" title="grafana dashboard" icon="chart-bar" tag="localhost:4000" >}}
  {{< card link="http://localhost:12345" title="alloy collector" icon="filter" tag="localhost:12345" >}}
{{< /cards >}}
