# ledger

The **ledger** service keeps track of each user's balance and transaction history with
**GoldenVCR Fun Points**. These points may be redeemed to perform certain actions within
the Golden VCR platform, such as triggering user-customized alerts during streams.

GoldenVCR Fun Points have no monetary value and are non-transferable.

- **OpenAPI specification:** https://golden-vcr.github.io/ledger/

## Development Guide

On a Linux or WSL system:

1. Install [Go 1.21](https://go.dev/doc/install)
2. Clone the [**terraform**](https://github.com/golden-vcr/terraform) repo alongside
   this one, and from the root of that repo:
    - Ensure that the module is initialized (via `terraform init`)
    - Ensure that valid terraform state is present
    - Run `terraform output -raw env_ledger_local > ../ledger/.env` to populate an
      `.env` file.
    - Run [`./local-db.sh up`](https://github.com/golden-vcr/terraform/blob/main/local-db.sh)
      to ensure that a Postgres server is running locally (requires
      [Docker](https://docs.docker.com/engine/install/)).
3. Ensure that the [**auth**](https://github.com/golden-vcr/auth?tab=readme-ov-file#development-guide)
   server is running locally.
4. From the root of this repository:
    - Run [`./db-migrate.sh`](./db-migrate.sh) to apply database migrations.
    - Run [`go run cmd/server/main.go`](./cmd/server/main.go) to start up the server.

Once done, the ledger server will be running at http://localhost:5003.

### Generating database queries

If you modify the SQL code in [`db/queries`](./db/queries/), you'll need to generate
new Go code to [`gen/queries`](./gen/queries/). To do so, simply run:

- `./db-generate-queries.sh`
