# Handmade Network

This is the codebase for the Handmade Network website, located online at https://handmade.network. The website is the home for all Handmade projects, the forums, the podcast, and other various initiatives.

The site is written in Go, and uses the standard library HTTP server with custom utilities for routing and middleware. It uses Postgres for its database, making heavy use of the excellent [pgx](https://github.com/jackc/pgx) library along with some custom wrappers. See the documentation in the `db` package for more details.

We want the website to be a great example of Handmade software on the web. We encourage you to read through the source, run it yourself, and learn from how we work. If you have questions about the code, or you're interested in contributing directly, reach out to us in #network-meta on the [Handmade Network Discord](https://discord.gg/hmn)!

## Prerequisites

You will need the following software installed:

- Go 1.22 or newer: https://go.dev/

    You can download Go directly from the website, or install it through major package managers. If you already have Go installed, but are unsure of the version, you can check by running `go version`.

- Postgres: https://www.postgresql.org/

    Any Postgres installation should work fine, although less common distributions may not work as nicely with our scripts out of the box. On Mac, [Postgres.app](https://postgresapp.com/) is recommended.

## First-time setup

- **Configure the site.** Copy `src/config/config.go.example` to `src/config/config.go`:

    ```
    # On Windows
    copy src\config\config.go.example src\config\config.go

    # On Mac and Linux
    cp src/config/config.go.example src/config/config.go
    ```

    Depending on your installation of Postgres, you may need to modify the hostname and port in the Postgres section of the config.

- **Set up the database.** Run `go run . db seed --create-user` to initialize the database and fill it with sample data.

- **Update your hosts file.** The website uses subdomains for official projects, so the site cannot simply be run off `localhost`. Add the following
line to your hosts file:

    ```
    127.0.0.1 handmade.local hero.handmade.local 4coder.handmade.local
    ```

    You may need to edit the hosts file again in the future if you add more official projects.

    On Windows, the hosts file is located at `C:\Windows\System32\Drivers\etc\hosts`. On Mac and Linux, the hosts file should be located at `/etc/hosts`. It can be edited using any plain text editor, but you will need superuser permissions.

## Running the site

Running the site is easy:

```
go run .
```

You should now be able to visit http://handmade.local:9001 to see the website!

There are also several other commands built into the website executable. You can see documentation for each of them by running `go run . help` or adding the `-h` flag to any command.

## Running tests

Also easy:

```
go test ./...
```

Note that we have a special coverage requirement for the `hmnurl` package. We use the tests in this package to ensure that our URL builder functions never go out of sync with the regexes used for routing. As a result, we mandate 100% coverage for all `Build` functions in `hmnurl`.

## Starter data

The `db seed` command pre-populates the site with realistic data. It also includes several user accounts that you may log into to test various situations:

| Username | Password |
| -------- | -------- |
| `admin` | `password` |
| `alice` | `password` |
| `bob` | `password` |
| `charlie` | `password` |

The `admin` user is a superuser on the site and will have access to all features, as well as the special admin UI for performing site maintenance and moderation. The other users are all normal users.
