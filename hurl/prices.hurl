POST http://localhost:8080/api/v1/instruments
Content-Type: application/json
{"kind":"stock","symbol":"PRICE","name":"Price observation scenario","quoteCurrency":"USD"}
HTTP 201
[Captures]
instrument_id: jsonpath "$.id"

GET http://localhost:8080/api/v1/instruments/{{instrument_id}}/prices
HTTP 200
[Asserts]
jsonpath "$.observations" count == 0

POST http://localhost:8080/api/v1/instruments/{{instrument_id}}/prices
Content-Type: application/json
{"amount":"00123.4","observedAt":"2026-09-14T12:00:00.123456789+08:00"}
HTTP 201
[Asserts]
jsonpath "$.instrumentId" == "{{instrument_id}}"
jsonpath "$.currency" == "USD"
jsonpath "$.amount" == "123.40"
jsonpath "$.observedAt" == "2026-09-14T04:00:00.123456789Z"

POST http://localhost:8080/api/v1/instruments/{{instrument_id}}/prices
Content-Type: application/json
{"amount":"0","observedAt":"2026-09-13T00:00:00Z"}
HTTP 201
[Asserts]
jsonpath "$.amount" == "0.00"

POST http://localhost:8080/api/v1/instruments/{{instrument_id}}/prices
Content-Type: application/json
{"amount":"0","observedAt":"2026-09-13T00:00:00Z"}
HTTP 201

POST http://localhost:8080/api/v1/instruments/{{instrument_id}}/prices
Content-Type: application/json
{"amount":"92233720368547758.07","observedAt":"2026-09-15T00:00:00Z"}
HTTP 201
[Asserts]
jsonpath "$.currency" == "USD"
jsonpath "$.amount" == "92233720368547758.07"

POST http://localhost:8080/api/v1/instruments/{{instrument_id}}/prices
Content-Type: application/json
{"amount":"92233720368547758.08","observedAt":"2026-09-14T00:00:00Z"}
HTTP 400
[Asserts]
body == "invalid amount\n"

POST http://localhost:8080/api/v1/instruments/{{instrument_id}}/prices
Content-Type: application/json
{"amount":"1","observedAt":"2026-09-14"}
HTTP 400
[Asserts]
body == "invalid observation timestamp\n"

GET http://localhost:8080/api/v1/instruments/{{instrument_id}}/prices
HTTP 200
[Asserts]
jsonpath "$.observations" count == 4
jsonpath "$.observations[0].amount" == "0.00"
jsonpath "$.observations[1].amount" == "0.00"
jsonpath "$.observations[2].amount" == "123.40"
jsonpath "$.observations[3].amount" == "92233720368547758.07"

GET http://localhost:8080/api/v1/instruments/not-a-uuid/prices
HTTP 400
[Asserts]
body == "invalid instrument ID\n"

GET http://localhost:8080/api/v1/instruments/00000000-0000-0000-0000-000000000000/prices
HTTP 404
[Asserts]
body == "instrument not found\n"

POST http://localhost:8080/api/v1/instruments/00000000-0000-0000-0000-000000000000/prices
Content-Type: application/json
{"amount":"1","observedAt":"2026-09-14T00:00:00Z"}
HTTP 404
[Asserts]
body == "instrument not found\n"

POST http://localhost:8080/api/v1/instruments/{{instrument_id}}/prices
Content-Type: application/json
{"amount":"42"}
HTTP 201
[Captures]
server_observed_at: jsonpath "$.observedAt"
[Asserts]
jsonpath "$.currency" == "USD"
jsonpath "$.amount" == "42.00"
jsonpath "$.observedAt" matches "^.*Z$"

GET http://localhost:8080/api/v1/instruments/{{instrument_id}}/prices
HTTP 200
[Asserts]
jsonpath "$.observations" count == 5
jsonpath "$.observations[*].observedAt" contains "{{server_observed_at}}"

POST http://localhost:8080/api/v1/instruments
Content-Type: application/json
{"kind":"stock","symbol":"VNDPRICE","name":"VND price scenario","quoteCurrency":"VND"}
HTTP 201
[Captures]
vnd_instrument_id: jsonpath "$.id"

POST http://localhost:8080/api/v1/instruments/{{vnd_instrument_id}}/prices
Content-Type: application/json
{"amount":"9223372036854775807"}
HTTP 201
[Asserts]
jsonpath "$.currency" == "VND"
jsonpath "$.amount" == "9223372036854775807"

POST http://localhost:8080/api/v1/instruments/{{vnd_instrument_id}}/prices
Content-Type: application/json
{"amount":"1.5"}
HTTP 400
[Asserts]
body == "invalid amount\n"
