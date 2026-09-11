# Create an account to own the balances in this scenario.
POST http://localhost:8080/api/v1/accounts
Content-Type: application/json
{
  "name": "  Hurl Cash Account  "
}
HTTP 201
[Captures]
account_id: jsonpath "$.id"
[Asserts]
jsonpath "$.name" == "Hurl Cash Account"

# Set a USD balance.
PUT http://localhost:8080/api/v1/accounts/{{account_id}}/cash/USD
Content-Type: application/json
{
  "amount": "1000"
}
HTTP 200
[Asserts]
header "Content-Type" startsWith "application/json"
jsonpath "$.currency" == "USD"
jsonpath "$.amount" == "1000.00"

# Set a VND balance with zero fractional digits.
PUT http://localhost:8080/api/v1/accounts/{{account_id}}/cash/VND
Content-Type: application/json
{
  "amount": "2500000"
}
HTTP 200
[Asserts]
jsonpath "$.currency" == "VND"
jsonpath "$.amount" == "2500000"

# Replacing USD is idempotent for the account/currency pair.
PUT http://localhost:8080/api/v1/accounts/{{account_id}}/cash/USD
Content-Type: application/json
{
  "amount": "1250.50"
}
HTTP 200
[Asserts]
jsonpath "$.currency" == "USD"
jsonpath "$.amount" == "1250.50"

# List all balances in deterministic currency order.
GET http://localhost:8080/api/v1/accounts/{{account_id}}/cash
HTTP 200
[Asserts]
header "Content-Type" startsWith "application/json"
jsonpath "$.balances[0].currency" == "USD"
jsonpath "$.balances[0].amount" == "1250.50"
jsonpath "$.balances[1].currency" == "VND"
jsonpath "$.balances[1].amount" == "2500000"
