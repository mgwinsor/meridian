GET http://localhost:8080/api/v1/instruments
HTTP 200
[Asserts]
jsonpath "$.instruments" exists

POST http://localhost:8080/api/v1/instruments
Content-Type: application/json
{
  "kind": "stock",
  "symbol": " AAPL ",
  "name": " Apple Inc. ",
  "quoteCurrency": "USD"
}
HTTP 201
[Captures]
instrument_id: jsonpath "$.id"
[Asserts]
jsonpath "$.kind" == "stock"
jsonpath "$.symbol" == "AAPL"
jsonpath "$.name" == "Apple Inc."
jsonpath "$.quoteCurrency" == "USD"
jsonpath "$.price" not exists
jsonpath "$.position" not exists

GET http://localhost:8080/api/v1/instruments/{{instrument_id}}
HTTP 200
[Asserts]
jsonpath "$.id" == "{{instrument_id}}"
jsonpath "$.symbol" == "AAPL"

GET http://localhost:8080/api/v1/instruments
HTTP 200
[Asserts]
jsonpath "$.instruments[*].id" contains "{{instrument_id}}"

POST http://localhost:8080/api/v1/instruments
Content-Type: application/json
{"kind":"stock","symbol":" ","name":"Apple","quoteCurrency":"USD"}
HTTP 400
[Asserts]
body == "instrument symbol cannot be empty\n"

POST http://localhost:8080/api/v1/instruments
Content-Type: application/json
{"kind":"unknown","symbol":"AAPL","name":"Apple","quoteCurrency":"USD"}
HTTP 400
[Asserts]
body == "unsupported instrument kind\n"

POST http://localhost:8080/api/v1/instruments
Content-Type: application/json
{"kind":"stock","symbol":"AAPL","name":"Apple","quoteCurrency":"EUR"}
HTTP 400
[Asserts]
body == "unsupported currency\n"

GET http://localhost:8080/api/v1/instruments/not-a-uuid
HTTP 400

GET http://localhost:8080/api/v1/instruments/00000000-0000-0000-0000-000000000000
HTTP 404
[Asserts]
body == "instrument not found\n"
