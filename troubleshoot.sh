#!/usr/bin/env bash

kubectl logs -l app=resource-collector -n webbai --tail -1 > webbai.log
