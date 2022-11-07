# bdog

Automatic API generation using reverse introspection of a database schema. e.g. point this software at your database and it will generate a RESTful API (no SQL knowledge required for usage).

## What does that mean?

For each table in your database, you will get an API endpoint. e.g. if you have a "products" table you will be able to access it using a `/products/` endpoint prefix. `bdog` will automatically create RESTful routes to CRUD (Create/Read/Update/Delete) using the primary key of your table (e.g. for a integer primary key, GET/PUT/DELETE on `/products/1234` would work as expected). The results of each request will contain JSON objects matching the column names from the database.

In addition to primary keys, foreign keys can also be used to link to and pull in related data for the JSON reponse. For example, given a foreign key from orders.product_id to products.product_id, a GET request to `/orders/5678?include=products` would include a nested object containing the data from the related products table!

## Quick Start Guide

To get started, find a database you want to play with, then build and run the `webapi` command pointing at that database. Here's a quick-start using the [airports example](./examples/README.md) data we've put together:

    cd examples
    bash fetch_airports.sh
    cd ../cmd/webapi
    go build .
    ./webapi ../../examples/airports.sqlite

## Current/MVP TODO list

- [x] Create list endpoints for each table
- [x] - Add Simple WHERE Filters
- [x] - Add Pagination
- [ ] - Determine best default ordering (are there date columns or a numeric PK?)
- [x] Create GET endpoints for rows in each table (using PK)
- [x] - Fetch row by column in unique index
- [x] - Nest data from linked table using foreign key (/orders/1234?include=customers)
- [x] - List related data from linked table using foreign key (/orders/1234/products)
- [x] Create PUT endpoint to update a row in a table
- [x] Create POST endpoint to create a row in a table
- [x] Create DELETE endpoint to delete a row in a table
- [x] Support Cross-Origin Resource Sharing (CORS) by default.
- [ ] Automatically expose OpenAPI definition of all endpoints
- [ ] - Include comments from database schema within OpenAPI spec.
- [ ] - Include example values for low-cardinality columns
- [ ] Create a basic authorization flow using Bearer Tokens (no expiration)

## Future/Magic stuff TODO list

- [x] Allow multi-column primary keys
- [ ] - Automatically order multi-column PKs by cardinality (e.g. for a vehicle use Year, then Make, then Model, etc. since there are fewer unique values for each in order)
- [ ] Determine many-to-many linking tables and hide them automatically
- [ ] Automatically determine low-cardinality columns and small fixed tables for enumerations
- [ ] Automatically create validation logic for create/update based on current data values
- [ ] If a deleted_at column exists, use soft delete logic throughout the API (per table)
