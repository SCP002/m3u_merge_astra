#!/bin/bash

clear
go clean -testcache
go test -v -timeout 10m -race -run ^TestRemoveDeadInputs$ m3u-merge-astra/astra
