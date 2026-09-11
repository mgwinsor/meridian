# Create an account and verify name normalization.
POST http://localhost:8080/api/v1/accounts
Content-Type: application/json
{
  "name": "  Hurl Primary  "
}
HTTP 201
[Captures]
account_id: jsonpath "$.id"
[Asserts]
header "Content-Type" startsWith "application/json"
jsonpath "$.id" matches "^[0-9a-f-]{36}$"
jsonpath "$.name" == "Hurl Primary"

# Retrieve the account created above.
GET http://localhost:8080/api/v1/accounts/{{account_id}}
HTTP 200
[Asserts]
header "Content-Type" startsWith "application/json"
jsonpath "$.id" == "{{account_id}}"
jsonpath "$.name" == "Hurl Primary"
