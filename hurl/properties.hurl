GET http://localhost:8080/api/v1/properties
HTTP 200
[Asserts]
jsonpath "$.properties" count == 0

POST http://localhost:8080/api/v1/properties
Content-Type: application/json
{"name":"  Singapore home  ","value":{"currency":"SGD","amount":" 0750000.5 "}}
HTTP 201
[Captures]
property_id: jsonpath "$.id"
[Asserts]
header "Content-Type" == "application/json"
header "Location" not exists
jsonpath "$.name" == "Singapore home"
jsonpath "$.value.currency" == "SGD"
jsonpath "$.value.amount" == "750000.50"

PUT http://localhost:8080/api/v1/properties/{{property_id}}/value
Content-Type: application/json
{"currency":"VND","amount":"9223372036854775807"}
HTTP 200
[Asserts]
jsonpath "$.id" == "{{property_id}}"
jsonpath "$.name" == "Singapore home"
jsonpath "$.value.currency" == "VND"
jsonpath "$.value.amount" == "9223372036854775807"

PUT http://localhost:8080/api/v1/properties/{{property_id}}/value
Content-Type: application/json
{"currency":"USD","amount":"92233720368547758.08"}
HTTP 400
[Asserts]
body == "invalid amount\n"

PUT http://localhost:8080/api/v1/properties/{{property_id}}/value
Content-Type: application/json
{"currency":"USD","amount":"0"}
HTTP 200
[Asserts]
jsonpath "$.value.amount" == "0.00"

GET http://localhost:8080/api/v1/properties
HTTP 200
[Asserts]
jsonpath "$.properties" count == 1
jsonpath "$.properties[0].id" == "{{property_id}}"
jsonpath "$.properties[0].value.currency" == "USD"
jsonpath "$.properties[0].value.amount" == "0.00"

PUT http://localhost:8080/api/v1/properties/00000000-0000-0000-0000-000000000000/value
Content-Type: application/json
{"currency":"USD","amount":"1"}
HTTP 404
[Asserts]
body == "property not found\n"
