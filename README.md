## Introduction:
Package sqldb provides tooling for establishing and managing a database connection, deploying and updating schemas, and running queries. This wraps the `sql` package to reduce some boilerplate while providing tools for schema and query management.

## Details:
- Supports MySQL, MariaDB, and SQLite. Additional database support should be relatively easy to add.
- Define a database connection configuration and manage the connection pool for running queries.
- Deploy and update database schema.
- Works either by storing the connection details within the package in a global manner or you can store the details separately elsewhere in your app. Storing the details separately allows for multiple different databases to be connected to at the same time.

## Getting Started:
You first need to define a configuration using `New...Config` or `Default...Config` based on if you want to store the configuration yourself or store it within this package. You can also define the configuration yourself using `sqldb.Config{}`.

Once you have a configuration defined, you can connect to your database using `Connect()` which will validate your
configuration and then try connecting. If you want to deploy or update your database schema, call `DeploySchema()` or `UpdateSchema()` *before* connecting.

Now with an established connection, run queries using your config in a manner such as `myconfig.Connection().Exec()`.

## Deploying and Updating Schema:
This works by providing a list of queries to run in your config. When the appropriate function is called, the queries will be run in order.

Queries used to deploy a database can optionally be translated from one database format to another (i.e.: MySQL to SQLite). This is useful since different database types structure their CREATE TABLE queries slightly differently but it would be a lot of extra work to maintain a separate set of queries for each database type. This works by doing a string replacement, and some more modifications, to queries so that you can write the queries in one database's format and still deploy to multiple database types.

Queries used to update the database are checked for safely ignorable errors. This is useful for instances where you rerun the `UpdateSchema` function (think, on each app start to ensure the schema is up to date) and want to ignore errors such as when a column already exists or was already removed.
