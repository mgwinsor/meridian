# Workspace totals may include existing data; the exact fixture is asserted in isolated Go tests.
POST http://localhost:8080/api/v1/accounts
Content-Type: application/json
{"name":"Overview fixture"}
HTTP 201
[Captures]
account_id: jsonpath "$.id"

PUT http://localhost:8080/api/v1/accounts/{{account_id}}/cash/USD
Content-Type: application/json
{"amount":"100"}
HTTP 200

POST http://localhost:8080/api/v1/instruments
Content-Type: application/json
{"kind":"stock","symbol":"OVERVIEW","name":"Overview equity","quoteCurrency":"SGD"}
HTTP 201
[Captures]
instrument_id: jsonpath "$.id"
[Asserts]
jsonpath "$.assetClass" == "equities"

PUT http://localhost:8080/api/v1/instruments/{{instrument_id}}/asset-class
Content-Type: application/json
{"assetClass":"mixed"}
HTTP 200
[Asserts]
jsonpath "$.assetClass" == "mixed"
jsonpath "$.symbol" == "OVERVIEW"
jsonpath "$.quoteCurrency" == "SGD"

PUT http://localhost:8080/api/v1/accounts/{{account_id}}/positions/{{instrument_id}}
Content-Type: application/json
{"quantity":"2"}
HTTP 200

GET http://localhost:8080/api/v1/overview
HTTP 200
[Asserts]
jsonpath "$.currency" == "USD"
jsonpath "$.complete" == false
jsonpath "$.assets[*].missingInputs[*]" contains "price"
jsonpath "$.assetClasses" exists
jsonpath "$.currencies" exists
jsonpath "$.fxObservations" exists

POST http://localhost:8080/api/v1/instruments/{{instrument_id}}/prices
Content-Type: application/json
{"amount":"10"}
HTTP 201

POST http://localhost:8080/api/v1/fx-rates/SGD
Content-Type: application/json
{"rate":"2","observedAt":"2020-01-01T08:00:00.123456789+08:00"}
HTTP 201
[Asserts]
jsonpath "$.rate" == "2"
jsonpath "$.observedAt" == "2020-01-01T00:00:00.123456Z"

POST http://localhost:8080/api/v1/fx-rates/VND
Content-Type: application/json
{"rate":"20000.123456789123456789"}
HTTP 201
[Asserts]
jsonpath "$.rate" == "20000.123456789123456789"

GET http://localhost:8080/api/v1/fx-rates/SGD
HTTP 200
[Asserts]
jsonpath "$.observations[*].rate" contains "2"

GET http://localhost:8080/api/v1/overview?currency=SGD
HTTP 200
[Asserts]
jsonpath "$.currency" == "SGD"
jsonpath "$.assets[*].convertedValue" contains "20.00"

GET http://localhost:8080/api/v1/overview?currency=VND
HTTP 200
[Asserts]
jsonpath "$.currency" == "VND"

GET http://localhost:8080/api/v1/overview?currency=EUR
HTTP 400

GET http://localhost:8080/api/v1/fx-rates/USD
HTTP 400

POST http://localhost:8080/api/v1/fx-rates/SGD
Content-Type: application/json
{"rate":"0"}
HTTP 400

POST http://localhost:8080/api/v1/fx-rates/VND
Content-Type: application/json
{"rate":"0.0000000000000000001"}
HTTP 400

POST http://localhost:8080/api/v1/fx-rates/SGD
Content-Type: application/json
{"rate":"2","observedAt":"2026-01-01T00:00:00"}
HTTP 400

PUT http://localhost:8080/api/v1/instruments/{{instrument_id}}/asset-class
Content-Type: application/json
{"assetClass":"cash"}
HTTP 400
