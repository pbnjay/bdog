git clone git@github.com:davidmegginson/ourairports-data.git

cd ourairports-data
sqlite3 airports.sqlite < ../basic_airports_schema.sql
mv airports.sqlite ..
cd ..

#; rm -fR ourairports-data