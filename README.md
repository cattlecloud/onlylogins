# OnlyLogins

This application powers [onlylogins](https://onlylogins.net).

### Development

#### Running locally

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

#### Accessing the database

```shell
# Open the SQLite file
just db-open ./hack/dev.toml

# Completely delete the SQLite file
just db-reset ./hack/dev.toml
```

#### Using memcached

A [memcached](https://www.memcached.org) instance is expected to be available
and is used for storing session state. For local development a Justfile target
exists for convenience, and works with the `hack/dev.toml` configuration.

```shell
just mc-run
```

#### Stripe integration

Note: it is not necessary to integrate with Stripe for local development or for
running your own instance. If the stripe configuration is left unset, `onlylogins`
will automatically exclude the subscription features.

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

### License

OnlyLogins is made available under the [AGPL-3.0](LICENSE) license.

The AGPL-3.0 is a strong copyleft license that imposes obligations materially
broader than those of permissive licenses. Any use, reproduction, modification,
or distribution of the OnlyLogins source code - including making a modified
version available to third parties over a network - is subject to the terms and
conditions of that license, and such terms should be reviewed in full prior to
any such activity.

CattleCloud LLC, as the copyright holder and owner of this repository, reserves
all rights not expressly granted under the AGPL-3.0. No additional permission
or exception is granted, whether by implication, estoppel, or otherwise, and
CattleCloud LLC intends to enforce the terms of the license to the fullest
extent permitted by applicable law.
