# TLS material

CDD serves HTTPS only and refuses to start without a certificate and a private
key, and it calls the Identity Server over HTTPS too, so this directory holds
both sides:

```
server.crt             the certificate CDD presents, PEM
server.key             its private key, PEM, mode 600
trust_store.pem        the Identity Server certificates CDD trusts when it calls them, PEM
client_trust_store.pem the client certificates CDD accepts on its sync endpoints, PEM
```

`tls.cert_dir` in `repository/conf/deployment.yaml` points here. A relative
path is resolved against the CDD home.

## Mutual TLS

As the pack ships, TLS is mutual in both directions, one key for each:

- `tls.mtls_enabled: true`: CDD presents `server.crt` as its client
  certificate when it calls the Identity Server. Set `tls.outbound_cert` and
  `tls.outbound_key` to present another key pair. Either way the certificate
  needs the TLS client authentication usage, which some public CAs no longer
  issue.
- `tls.require_client_cert: true`: the sync endpoints the Identity Server
  calls refuse a request without a client certificate that
  `client_trust_store.pem` verifies, along with the sync credentials. The
  endpoints browsers and SDKs call ask for none. The Identity Server's CDD
  connector presents the key pair of its primary keystore; a connector too old
  to present one gets 401 on every sync.

`client_trust_store.pem` names who may call the sync endpoints. Put the
Identity Server's own certificate in it, or a CA that issues only to the
Identity Server. Never put a public CA or a shared corporate CA in it: every
certificate that CA issues would be accepted.

Turn `require_client_cert` off only where something in front of CDD ends the
TLS, such as a load balancer. The client certificate does not reach CDD there.

## What this pack ships

All four files are **Identity Server 7.3.0's default key pair**: the
`wso2carbon` entry of its `wso2carbon.p12`. A default Identity Server 7.3.0
presents that key pair and already trusts its certificate in its
`client-truststore.p12`. CDD presents it and trusts it here, so the two call
each other over mutual TLS with no certificate set up on either side.

That holds only while they reach each other as **`localhost`**: the
certificate is issued for `localhost` and nothing else. It suits a first
installation with both on one host. Containers, separate hosts, or any other
name need the certificates below.

## Replace it for production

**This key pair is a convenience for getting started, not a credential.** Its
private key is in every Identity Server 7.3.0 download, so anyone can present
themselves as this CDD, or as that Identity Server. `cdd start` warns while
it is in use, here or in `client_trust_store.pem`.

Replace it on both sides:

1. Here: `server.crt` and `server.key` with a certificate issued for the host
   in `server_url`, the name the Identity Server and the Console reach CDD on.
2. Here: `trust_store.pem` with the certificate the Identity Server presents,
   or the CA that issued it.
3. Here: `client_trust_store.pem` with the certificate the Identity Server
   presents as its client certificate: the one in its primary keystore.
4. On the Identity Server: import the new `server.crt`, or its CA, into
   `client-truststore.p12`, and restart it.

In a container, mount the files rather than changing the image: this directory
for CDD, and the truststore for the Identity Server.
