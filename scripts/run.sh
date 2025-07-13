#!/bin/bash

set -e

go run cmd/migrations/main.go
go run cmd/api/main.go
