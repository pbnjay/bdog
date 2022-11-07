# Airports Example data set

This example data set uses sources from the OurAirports Data, graciously and publically released here: https://ourairports.com/data/

## Reproducing this data set

We've provided a shell script (`fetch_airports.sh`) to download and load the data above into a sqlite database. We fetch the CSV files from the github repo linked above, and then apply a simple sqlite3 schema and loading script (found at `basic_airports_schema.sql`)

## Available APIs

Using this data with bdog is as easy as running `./webapi airports.sqlite` which gives the following default API endpoints:

     GET /countries/
     GET /countries/:code
     GET /countries/:code/regions
     GET /countries/:code/airports
     POST /countries
     PUT /countries/:code
     DELETE /countries/:code

     GET /regions/
     GET /regions/:code
     GET /regions/:code?include=countries
     GET /regions/:code/airports
     POST /regions
     PUT /regions/:code
     DELETE /regions/:code

     GET /airports/
     GET /airports/:ident
     GET /airports/:ident?include=regions
     GET /airports/:ident?include=countries
     POST /airports
     PUT /airports/:ident
     DELETE /airports/:ident

## Usage examples:

List the first page of countries in the database:

    $ curl http://127.0.0.1:8080/countries
    [
      {
        "code": "AD",
        "continent": "EU",
        "keywords": "Andorran airports",
        "name": "Andorra",
        "wikipedia_link": "https://en.wikipedia.org/wiki/Andorra"
      },
      ... omitted for brevity ...
      {
        "code": "AR",
        "continent": "SA",
        "keywords": "Aeropuertos de Argentina",
        "name": "Argentina",
        "wikipedia_link": "https://en.wikipedia.org/wiki/Argentina"
      }
    ]

    # list the next page
    $ curl http://127.0.0.1:8080/countries?_page=2
    ...

Fetch country information by `code` (abbreviation):

    $ curl http://127.0.0.1:8080/countries/US
    {
      "code": "US",
      "continent": "NA",
      "keywords": "American airports",
      "name": "United States",
      "wikipedia_link": "https://en.wikipedia.org/wiki/United_States"
    }

Update the name of a country:

    $ curl -X PUT -d name="United States of America" http://127.0.0.1:8080/countries/US
    {
      "code": "US",
      "continent": "NA",
      "keywords": "American airports",
      "name": "United States of America",
      "wikipedia_link": "https://en.wikipedia.org/wiki/United_States"
    }

List the first page of regions of a country:

    $ curl http://127.0.0.1:8080/countries/US/regions
    [
      {
        "code": "US-AK",
        "continent": "NA",
        "iso_country": "US",
        "keywords": "Airports in Alaska",
        "local_code": "AK",
        "name": "Alaska",
        "wikipedia_link": "https://en.wikipedia.org/wiki/Alaska"
      },
      ... omitted for brevity ...
      {
        "code": "US-FL",
        "continent": "NA",
        "iso_country": "US",
        "keywords": "Airports in Florida",
        "local_code": "FL",
        "name": "Florida",
        "wikipedia_link": "https://en.wikipedia.org/wiki/Florida"
      }
    ]

List the next page:

    $ curl http://127.0.0.1:8080/countries/US/regions?_page=2
    ...

Get details about a region, and include nested information for the linked `countries` table: (TODO: automatically map from plural to singular)

    $ curl http://127.0.0.1:8080/regions/US-NC?include=countries
    {
      "code": "US-NC",
      "continent": "NA",
      "countries": {
        "code": "US",
        "continent": "NA",
        "keywords": "American airports",
        "name": "United States of America",
        "wikipedia_link": "https://en.wikipedia.org/wiki/United_States"
      },
      "iso_country": "US",
      "keywords": "Airports in North Carolina",
      "local_code": "NC",
      "name": "North Carolina",
      "wikipedia_link": "https://en.wikipedia.org/wiki/North_Carolina"
    }

List the airports in region `US-NC`, filtered to those with `scheduled_service=yes` and `type=large_airport`

    $ curl 'http://127.0.0.1:8080/regions/US-NC/airports?scheduled_service=yes&type=large_airport'
    [
      {
        "continent": "NA",
        "elevation_ft": "748",
        "gps_code": "KCLT",
        "home_link": "http://www.charlotteairport.com/",
        "iata_code": "CLT",
        "ident": "KCLT",
        "iso_country": "US",
        "iso_region": "US-NC",
        "keywords": "",
        "latitude_deg": "35.2140007019043",
        "local_code": "CLT",
        "longitude_deg": "-80.94309997558594",
        "municipality": "Charlotte",
        "name": "Charlotte Douglas International Airport",
        "scheduled_service": "yes",
        "type": "large_airport",
        "wikipedia_link": "https://en.wikipedia.org/wiki/Charlotte/Douglas_International_Airport"
      },
      {
        "continent": "NA",
        "elevation_ft": "435",
        "gps_code": "KRDU",
        "home_link": "",
        "iata_code": "RDU",
        "ident": "KRDU",
        "iso_country": "US",
        "iso_region": "US-NC",
        "keywords": "",
        "latitude_deg": "35.877601623535156",
        "local_code": "RDU",
        "longitude_deg": "-78.7874984741211",
        "municipality": "Raleigh/Durham",
        "name": "Raleigh Durham International Airport",
        "scheduled_service": "yes",
        "type": "large_airport",
        "wikipedia_link": "https://en.wikipedia.org/wiki/Raleigh-Durham_International_Airport"
      }
    ]

UTF-8 codepoints are fully supported and returned by endpoints (see keywords)

    $ curl http://127.0.0.1:8080/countries/RU
    {
      "code": "RU",
      "continent": "EU",
      "keywords": "Soviet, Sovietskaya, Sovetskaya, Аэропорты России",
      "name": "Russia",
      "wikipedia_link": "https://en.wikipedia.org/wiki/Russia"
    }

The DELETE method also works as expected:

    $ curl -X DELETE http://127.0.0.1:8080/countries/RU
    {"message":"record successfully deleted"}

    $ curl http://127.0.0.1:8080/countries/RU
    Not Found

And the POST method will create entries:

    $ curl -X POST -H "Content-Type: application/json" -d '{"code":"RU","continent":"EU","keywords":"Soviet, Sovietskaya, Sovetskaya, Аэропорты России","name":"Russia","wikipedia_link":"https://en.wikipedia.org/wiki/Russia"}' http://127.0.0.1:8080/countries
    {
      "code": "RU",
      "continent": "EU",
      "keywords": "Soviet, Sovietskaya, Sovetskaya, Аэропорты России",
      "name": "Russia",
      "wikipedia_link": "https://en.wikipedia.org/wiki/Russia"
    }
