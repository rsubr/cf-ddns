# Cloudflare Dynamic DNS Client

`cf-ddns` is a standalone Go binary that updates your Cloudflare DNS record with your current public IPv4 address. This is useful if you want to host a home lab using a dynamic IP address and are hosting your DNS on Cloudflare. Note that IPv6 addresses are *not* supported.

`cf-ddns` follows the principle of least privilege and ensures that the Cloudflare credentials can be used to updated only the selected DNS record. You will not be exposing your entire DNS settings using a highly privileged global API key.

`cf-ddns` first checks if the DDNS IP is updated, and only then calls the Cloudflare API if required making it safe to run `cf-ddns` as a repeated cron job. The application is completely stateless and does not use any local storage.


## Using

Assume you want to update the DNS A record for `homelab.example.com` whenever your public IP changes.

### Preparing Cloudflare

1. Create DNS A record for `homelab.example.com` with a dummy default value and configure all the necessary Cloudflare settings for the record. `cf-ddns` will only update the IP address and not change any other setting for this record.

2. Generate two [User API Tokens](dash.cloudflare.com/profile/api-tokens) with the following permissions:
    - `DNS:Edit` for the zone containing the DNS record. eg. `example.com`
    - `DNS:Read` for the zone containing the DNS record. eg. `example.com`

3. Open the Overview Page for `example.com` and on the bottom right copy the `Zone ID`.

4. You will have the below parameters and secrets:

| Name  | Purpose |
|-------|---------|
| CLOUDFLARE_ZONE_ID | Cloudflare Zone ID |
| CLOUDFLARE_API_TOKEN | Cloudflare API Token with `DNS:Edit` permission |
| CLOUDFLARE_API_READ_TOKEN | Cloudflare API Token with `DNS:Read` permission |
| CLOUDFLARE_DNS_RECORD_NAME | DNS record to update. eg. `homelab.example.com` |

### Fetching the `dns_record_id`

1. Use the DNS:Read token to fetch the `dns_record_id` for `lab.example.com`. Review the returned JSON response and save the `id` key for `lab.example.com` as `CLOUDFLARE_DNS_RECORD_ID` as this will be used by `cf-ddns` to update the DNS record.

```bash
export CLOUDFLARE_ZONE_ID='ZONE_ID'
export CLOUDFLARE_API_READ_TOKEN='API_TOKEN'

curl -X GET "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records" \
     -H "Authorization: Bearer ${api_token}" \
     -H "Content-Type:application/json" | jq
```

### Running `cf-ddns`

1. Download the latest release and copy it to `/usr/local/sbin/cf-ddns`.

2. Create the below shell script to run `cf-ddns` binary with required environment vars.

```bash
#!/bin/bash
# Filename: /usr/local/sbin/cf-ddns.sh

export CLOUDFLARE_ZONE_ID='ZONE_ID'
export CLOUDFLARE_API_TOKEN='API_TOKEN'
export CLOUDFLARE_DNS_RECORD_NAME='homelab.example.com'
export CLOUDFLARE_DNS_RECORD_ID='DNS_RECORD_ID'

exec /usr/local/sbin/cf-ddns
```

3. Setup the following crontab entry to run `cf-ddns` every 5 minutes.

```cron
# Update CloudFlare DDNS records for lab.example.com every 5 mins
*/5 *   * * *   nobody /usr/local/sbin/cf-ddns.sh
```

## How Does This Work?

1. Get the public IP of the client using [Cloudflare Trace](https://cloudflare.com/cdn-cgi/trace).

2. Get the current IP address of the DNS record using [Cloudflare DoH](https://developers.cloudflare.com/1.1.1.1/encryption/dns-over-https/make-api-requests/dns-json/).

3. If the public IP and current IP are different, updated the DNS A record using the [Cloudflare API](https://developers.cloudflare.com/api/operations/dns-records-for-a-zone-patch-dns-record).

