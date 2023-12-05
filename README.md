# ledger

The **ledger** service keeps track of each user's balance and transaction history with
**GoldenVCR Fun Points**. These points may be redeemed to perform certain actions within
the Golden VCR platform, such as triggering user-customized alerts during streams.

GoldenVCR Fun Points have no monetary value and are non-transferable.

- **OpenAPI specification:** https://golden-vcr.github.io/ledger/

## Prerequisites

Install [Go 1.21](https://go.dev/doc/install). If successful, you should be able to run:

```
> go version
go version go1.21.0 windows/amd64
```

## Initial setup

Create a file in the root of this repo called `.env` that contains the environment
variables required in [`main.go`](./cmd/server/main.go). If you have the
[`terraform`](https://github.com/golden-vcr/terraform) repo cloned alongside this one,
simply open a shell there and run:

- `terraform output -raw twitch_api_env > ../ledger/.env`
- `terraform output -raw ledger_s2s_auth_env >> ../ledger/.env`
- `./local-db.sh env >> ../ledger/.env`

### Running the database

This API stores persistent data in a PostgreSQL database. When running in a live
environment, each API has its own database, and connection details are configured from
Terraform secrets via .env files.

For local development, we run a self-contained postgres database in Docker, and all
server-side applications share the same set of throwaway credentials.

We use a script in the [`terraform`](https://github.com/golden-vcr/terraform) repo,
called `./local-db.sh`, to manage this local database. To start up a fresh database and
apply migrations, run:

- _(from `terraform`:)_ `./local-db.sh up`
- _(from `ledger`:)_ `./db-migrate.sh`

If you need to blow away your local database and start over, just run
`./local-db.sh down` and repeat these steps.

### Generating database queries

If you modify the SQL code in [`db/queries`](./db/queries/), you'll need to generate
new Go code to [`gen/queries`](./gen/queries/). To do so, simply run:

- `./db-generate-queries.sh`

## Running

Once your `.env` file is populated, you should be able to build and run the server:

- `go run cmd/server/main.go`

If successful, you should be able to run `curl http://localhost:5003/status` and
receive a response.

## Auth dependency

Note that in order to call endpoints that require authorization, you'll need to be
running the [auth](https://github.com/golden-vcr/auth) API locally as well. By default,
**ledger** is configured to reach the auth server via its default URL of
`http://localhost:5002`, so it's sufficient to simply `go run cmd/server/main.go` from
both repos.

In production, **ledger** and **auth** currently run alongside each other on a single
DigitalOcean droplet, with auth listening on 5002, so no further configuration is
necessary. If we scale up beyond a single host, then showtime should be configured with
an appropriate `AUTH_URL` value to hit the production auth server.
