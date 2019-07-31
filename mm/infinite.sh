#!/bin/bash
while ! go run *.go
do
  sleep 1
  echo "Restarting program..."
done
