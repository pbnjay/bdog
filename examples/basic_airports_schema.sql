-- These tables match the CSV data files from OurAirports
-- Although for our purposes we will ignore the internal ids.
--
-- Since the original CSVs do not have primary keys, foreign
-- keys, or unique constraints we add them in to show off
-- some of the bdog features.

CREATE TABLE IF NOT EXISTS "countries"(
  "ignored_id" TEXT,
  "code" TEXT PRIMARY KEY,
  "name" TEXT UNIQUE,
  "continent" TEXT,
  "wikipedia_link" TEXT,
  "keywords" TEXT
);

CREATE TABLE IF NOT EXISTS "regions"(
  "ignored_id" TEXT,
  "code" TEXT PRIMARY KEY,
  "local_code" TEXT,
  "name" TEXT,
  "continent" TEXT,
  "iso_country" TEXT REFERENCES "countries" ("code"),
  "wikipedia_link" TEXT,
  "keywords" TEXT,
  UNIQUE ("iso_country", "local_code")
);

CREATE TABLE IF NOT EXISTS "airports"(
  "ignored_id" TEXT,
  "ident" TEXT PRIMARY KEY,
  "type" TEXT,
  "name" TEXT,
  "latitude_deg" REAL,
  "longitude_deg" REAL,
  "elevation_ft" INTEGER,
  "continent" TEXT,
  "iso_country" TEXT REFERENCES "countries" ("code"),
  "iso_region" TEXT REFERENCES "regions" ("code"),
  "municipality" TEXT,
  "scheduled_service" TEXT, -- "yes" or "no"
  "gps_code" TEXT,
  "iata_code" TEXT,
  "local_code" TEXT,
  "home_link" TEXT,
  "wikipedia_link" TEXT,
  "keywords" TEXT
);

-- these 4 lines are sqlite specific
.mode csv
.import --skip 1 countries.csv countries
.import --skip 1 regions.csv regions
.import --skip 1 airports.csv airports

alter table countries drop column ignored_id;
alter table regions drop column ignored_id;
alter table airports drop column ignored_id;