#!/bin/sh
if [ "$DEBUG" = "true" ]; then
    DLV_PORT=${DLV_PORT:-40000}
    dlv exec ./ai-chatbot --headless --listen=":$DLV_PORT" --api-version=2 --log --accept-multiclient  -- --config_file config_$ENV.yaml
else
    ./ai-chatbot --config_file config_$ENV.yaml
fi
