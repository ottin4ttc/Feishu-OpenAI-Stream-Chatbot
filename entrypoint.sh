#!/bin/sh
if [ "$DEBUG" = "true" ]; then
    DLV_PORT=${DLV_PORT:-40000}
    dlv exec ./user-service --headless --listen=":$DLV_PORT" --api-version=2 --log --accept-multiclient  -- --config_file config_$ENV.yaml
else
    ./user-service --config_file config_$ENV.yaml
fi
