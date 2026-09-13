POST http://localhost:8080/api/v1/accounts
Content-Type: application/json
{"name":"Position account"}
HTTP 201
[Captures]
account_id: jsonpath "$.id"

POST http://localhost:8080/api/v1/instruments
Content-Type: application/json
{"kind":"stock","symbol":"POS","name":"Position test","quoteCurrency":"USD"}
HTTP 201
[Captures]
instrument_id: jsonpath "$.id"

GET http://localhost:8080/api/v1/accounts/{{account_id}}/positions
HTTP 200
[Asserts]
jsonpath "$.positions" count == 0

PUT http://localhost:8080/api/v1/accounts/{{account_id}}/positions/{{instrument_id}}
Content-Type: application/json
{"quantity":"0012.3456789012345678900"}
HTTP 200
[Asserts]
jsonpath "$.accountId" == "{{account_id}}"
jsonpath "$.instrumentId" == "{{instrument_id}}"
jsonpath "$.quantity" == "12.34567890123456789"

PUT http://localhost:8080/api/v1/accounts/{{account_id}}/positions/{{instrument_id}}
Content-Type: application/json
{"quantity":"0"}
HTTP 200

PUT http://localhost:8080/api/v1/accounts/{{account_id}}/positions/{{instrument_id}}
Content-Type: application/json
{"quantity":"-1"}
HTTP 400
[Asserts]
body == "invalid quantity\n"

GET http://localhost:8080/api/v1/accounts/{{account_id}}/positions
HTTP 200
[Asserts]
jsonpath "$.positions" count == 1
jsonpath "$.positions[0].quantity" == "0"

PUT http://localhost:8080/api/v1/accounts/{{account_id}}/positions/00000000-0000-0000-0000-000000000000
Content-Type: application/json
{"quantity":"1"}
HTTP 404
[Asserts]
body == "instrument not found\n"
