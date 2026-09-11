# Invalid account JSON shape.
POST http://localhost:8080/api/v1/accounts
Content-Type: application/json
"not an object"
HTTP 400
[Asserts]
body contains "invalid request"

# Blank account names are rejected.
POST http://localhost:8080/api/v1/accounts
Content-Type: application/json
{
  "name": "   "
}
HTTP 400
[Asserts]
body contains "account name cannot be empty"

# Malformed account IDs are rejected at the HTTP boundary.
GET http://localhost:8080/api/v1/accounts/not-a-uuid
HTTP 400
[Asserts]
body contains "invalid account ID"

# Missing accounts return 404.
GET http://localhost:8080/api/v1/accounts/00000000-0000-0000-0000-000000000001
HTTP 404
[Asserts]
body contains "account not found"

# Unsupported account methods return 405.
DELETE http://localhost:8080/api/v1/accounts/00000000-0000-0000-0000-000000000001
HTTP 405

# Unsupported cash currencies are rejected.
PUT http://localhost:8080/api/v1/accounts/00000000-0000-0000-0000-000000000001/cash/GBP
Content-Type: application/json
{"amount":"1.00"}
HTTP 400
[Asserts]
body contains "unsupported currency"

# Negative and over-precise cash amounts are rejected.
PUT http://localhost:8080/api/v1/accounts/00000000-0000-0000-0000-000000000001/cash/USD
Content-Type: application/json
{"amount":"-1.00"}
HTTP 400
[Asserts]
body contains "invalid amount"

PUT http://localhost:8080/api/v1/accounts/00000000-0000-0000-0000-000000000001/cash/USD
Content-Type: application/json
{"amount":"1.001"}
HTTP 400
[Asserts]
body contains "invalid amount"

# Unsupported cash methods return 405.
DELETE http://localhost:8080/api/v1/accounts/00000000-0000-0000-0000-000000000001/cash/USD
HTTP 405
