# caddy-on-demand-tls-pg

The Caddy web server is super cool for a load of reasons. One super cool and unique feature it supports is [on-demand TLS](https://caddyserver.com/docs/automatic-https#on-demand-tls). This feature is fantastic for SaaS providers since it means they never need to touch Caddy in order to onboard customers. When a request comes in, if Caddy has never seen the domain before, it will ask the [HTTP server configured in the `on_demand_tls` configuration](https://caddyserver.com/docs/caddyfile/options#on-demand-tls) if it should allow the domain with the domain as a query parameter. If it gets a 2xx response, it quickly does the CA handshake.

This requires both a fast HTTP server that can get the response and a handler that can check if the domain exists. We write a lot of Ruby on Rails, and whilst it is plenty fast for most things, we want something extremely fast for something running in the middle of a handshake. We also realised this could be its own little open source library since it only has to go and check a database in our (and I suspect many other peoples) use cases. Hence, this library was born, using both fasthttp and pgx (libraries known for their sheer performance)!

## Configuring the hostname

By default, the application will run on `:8383`. This is fine for within Docker or Kubernetes since if you want to bind to the local loopback (you should if you are using this in a per host manor!) you just bind to `127.0.0.1:8383`. This might, however, be problematic if for whatever reason you are running the pure binary. To avoid this, just use the `HOST` variable to set it to `127.0.0.1:8383` (or whatever port you want) locally in this case.

## Building the config

To build a config, we will need to make a JSON object with the following items:

| Key          | Comments                                                                                                                                                                             |
|--------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| postgres_uri | The URI to use to connect to Postgres. Since we use pgxpool under the hood, this also handles connection pooling. You can use pool_max_conns in the query parameters to handle this. |
| one_of       | An array of {"table_name": string, "column": string} that will be checked as equal to the domain.                                                                                    |                                                                       |   |   |   |

> ⚠ **Both table_name and column are not sanitized!** We deemed this to not be a risk since the only people managing this deployment should only be the people managing the relationship between the web server and backend anyway, but it is something to be aware of. The `domain` sent to the database is sanitized.

This config can be then handed the application in one of two ways:

1. If you base64 encode it, you can pass it as the `CONFIG` environment variable.
2. The application will try reading `config.json` from the cwd. In the case of the Docker image, this means putting the configuration at `/app/config.json` will cause it to be read.

## Usage

Once you have configured your service and run it, you should see this:

```
🚀 Listening on [::]:8383
```

<p align="center">
    <i>(The hostname will be different if changed with the <code>HOST</code> variable)</i>
</p>

If you do, it works! The table columns you listed will be compared against any domains Caddy requests to your server. If it is found, the server will return a 204. If not, it will return a 404.

You can pass any path to this web server in your Caddy configuration, you might as well pass the root though.
