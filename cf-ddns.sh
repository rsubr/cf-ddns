#!/bin/bash
# Filename: /usr/local/sbin/cf-ddns.sh
#
# Add the below /etc/crontab entry to check periodically
# Update CloudFlare DDNS records for lab.example.com every 5 mins
# */5 *   * * *   nobody /usr/local/sbin/cf-ddns.sh

export CLOUDFLARE_ZONE_ID='ZONE_ID'
export CLOUDFLARE_API_TOKEN='API_TOKEN'
export CLOUDFLARE_DNS_RECORD_NAME='homelab.example.com'
export CLOUDFLARE_DNS_RECORD_ID='DNS_RECORD_ID'

exec /usr/local/sbin/cf-ddns
