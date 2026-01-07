#!/bin/bash

unikraft instance create \
	--set metro=ukp-stable \
	--set autostart=true \
	--set image=jedevc/www:latest \
	--set resources.memory=512 \
	--set resources.vcpus=1 \
	--set "runtime.args=/usr/bin/caddy run --config /etc/caddy/Caddyfile" \
	--set service.services=443:2015/tls+http \
	--set service.domains=name=jedevcwww
