# OnlyLogins

This application powers [onlylogins](https://onlylogins.net).

#### Stripe

1. download the [Stripe CLI](https://docs.stripe.com/stripe-cli/install?install-method=linux) tool

2. login using the tool

```
stripe login
```

3. run the tool to proxy

```
stripe listen --all-snapshot --forward-to localhost:3000/api/stripe/webhooks
```

4. Be sure to use the "-Sandbox" in stripe for managing test environments.
