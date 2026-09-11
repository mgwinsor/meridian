# Liveness remains available during normal operation.
GET http://localhost:8080/livez
HTTP 200
[Asserts]
header "Content-Type" startsWith "text/plain"
body == "OK"

# Readiness is available during normal operation.
GET http://localhost:8080/readyz
HTTP 200
[Asserts]
header "Content-Type" startsWith "text/plain"
body == "OK"
