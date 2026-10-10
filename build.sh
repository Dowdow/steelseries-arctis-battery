#!/bin/sh

set -e

CGO_ENABLED=0 go build -o dist/steelseries-arctis-battery

echo "Build completed successfully: dist/steelseries-arctis-battery"
