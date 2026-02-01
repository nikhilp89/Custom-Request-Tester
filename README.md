# Custom-Request-Tester

A concurrent HTTP testing tool written in Go that ingests YAML request templates and checks generated responses for specified regexes
 
# Overview

This tool enables security researchers and developers to:

Test multiple subdomains concurrently with custom HTTP requests
Define custom headers, methods, and request bodies via YAML configuration
Filter and grep responses by status code, headers, or body content
Save matching results to an output file
# Features

Concurrent Testing: Uses goroutines with configurable concurrency (default: 50 concurrent requests)
Flexible Request Configuration: Define requests via YAML with custom headers, methods, and bodies
Response Filtering: Filter results by:
Response body content (regex support)
Response headers (specific or all headers)
HTTP status codes
TLS Support: Handles HTTPS with configurable TLS verification
Timeout Management: Built-in request timeouts to prevent hanging
Output Management: Thread-safe file writing with mutex protection
# Installation

Prerequisites

Go 1.16 or higher
Required dependencies:
go get github.com/pieterclaerhout/go-waitgroup
go get gopkg.in/yaml.v3

# Build

go build -o subdomain-scanner main.go

# Usage

Basic Command

./subdomain-scanner -headers <request-config.yaml> -subdomains <subdomains.txt> [options]

Required Flags

Flag	Description
-headers	Path to YAML file containing request configuration
-subdomains	Path to text file containing list of subdomains (one per line)
Optional Flags

Flag	Default	Description
-grepLocation	default	Where to search: body, headers, statuscode, or default
-grepHeader	default	Specific header name to search (when grepLocation=headers)
-grep	test	Regex pattern to search for in responses
-grepStatusCode	0	Filter by HTTP status code (0 = any status code)
Configuration Files

Request Configuration (YAML)

Create a YAML file to define your HTTP request template:

request1:
  Method: GET
  Url: /api/v1/endpoint
  Protocol: HTTP/1.1
  Headers:
    - Name: Content-Type
      Value: application/json
    - Name: X-Custom-Header
      Value: custom-value
    - Name: Cookie
      Value: session=abc123
  Body: ""

With Request Body (POST/PUT):

request1:
  Method: POST
  Url: /api/login
  Protocol: HTTP/1.1
  Headers:
    - Name: Content-Type
      Value: application/json
  Body: '{"username":"test","password":"test123"}'

Subdomains File (Text)

Create a text file with one subdomain per line:

api.example.com
dev.example.com
staging.example.com
admin.example.com

# Usage Examples

Example 1: Find APIs Returning JSON

./subdomain-scanner \
  -headers request.yaml \
  -subdomains domains.txt \
  -grepLocation headers \
  -grepHeader Content-Type \
  -grep "application/json"

Example 2: Find 200 OK Responses

./subdomain-scanner \
  -headers request.yaml \
  -subdomains domains.txt \
  -grepLocation statuscode \
  -grepStatusCode 200

Example 3: Search Response Bodies for Secrets

./subdomain-scanner \
  -headers request.yaml \
  -subdomains domains.txt \
  -grepLocation body \
  -grep "api[_-]?key|secret|password"

Example 4: Find Specific Headers with Status Code

./subdomain-scanner \
  -headers request.yaml \
  -subdomains domains.txt \
  -grepLocation headers \
  -grepHeader Server \
  -grep "nginx" \
  -grepStatusCode 200

# Output

Results are written to a file named output in the current directory. The output format varies based on grep settings:

Status code filtering: http://subdomain.example.com/path
Header filtering: http://subdomain.example.com/path | header-value
Body filtering: http://subdomain.example.com/path
How It Works

Initialization: Parses command-line arguments and validates inputs
Configuration Loading: Reads YAML request configuration and subdomain list
Request Generation: Creates HTTP requests for each subdomain with custom headers/body
Concurrent Execution: Spawns goroutines (max 50 concurrent) to test all subdomains
Response Filtering: Applies grep patterns to filter interesting responses
Output Writing: Thread-safe writing of matched results to output file
Technical Details

# HTTP Client Configuration

TLS Verification: Disabled by default (InsecureSkipVerify: true)
Timeout: 3 seconds for basic requests, 10 seconds for body requests
Connection Pooling: Max 1000 idle connections
Redirect Handling: Follows redirects (up to 100) in generateRequest1
Concurrency

Uses go-waitgroup library for controlled concurrency
Default limit: 50 concurrent goroutines
Thread-safe file writing with mutex locks
Default Headers

Accept: */*
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36...
