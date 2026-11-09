# OnlyLogins

This application powers [onlylogins](https://onlylogins.net).

#### Local development

Assuming you have a working Go toolchain, getting started is as easy as running
a few [Justfile](https://just.systems/man/en/) targets.

```shell
# Install local tooling, run linters and tests
just init lint tests
```

```shell
# Run the instance and listen on localhost:3000
just run ./hack/dev.toml
```

Access or reset the database

```shell
# Open the SQLite file
just db-open ./hack/dev.toml

# Completely delete the SQLite file
just db-reset ./hack/dev.toml
```

#### Memcached

A [memcached](https://www.memcached.org) instance is expected to be available
and is used for storing session state. For local development a Justfile target
exists for convenience, and works with the `hack/dev.toml` configuration.

```shell
just mc-run
```

#### Stripe

1. download the [Stripe CLI](https://docs.stripe.com/stripe-cli/install?install-method=linux) tool

2. login using the tool

```shell
stripe login
```

3. run the tool to proxy

```shell
stripe listen --all-snapshot --forward-to localhost:3000/api/stripe/webhooks
```

4. Be sure to use the "-Sandbox" in stripe for managing test environments.
