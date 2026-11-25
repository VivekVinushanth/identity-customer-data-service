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

The Identity Server calls the CDD sync endpoints, and CDD calls the Identity
Server's APIs. Only the sync endpoints are private to the two, so that is where
CDD requires mutual TLS:

- `tls.require_client_cert: true`: the sync endpoints refuse a request without
  a client certificate that `client_trust_store.pem` verifies, along with the
  sync credentials. The endpoints browsers and SDKs call ask for none. The
  Identity Server's CDD connector presents the key pair of
  `[customer_data_service.client_certificate]` in its `deployment.toml`, or of
  its primary keystore when that is not set; a connector too old to present
  one gets 401 on every sync.
- `tls.mtls_enabled: true`: CDD presents a client certificate when it calls
  the Identity Server: `tls.outbound_cert` and `tls.outbound_key` when they are
  set, `server.crt` otherwise. The Identity Server does not require it: CDD
  authenticates there with OAuth client credentials.

A client certificate needs the TLS client authentication usage (extended key
usage `clientAuth`), or no extended key usage at all. Certificates from public
CAs are now for TLS server authentication only, and both CDD and the Identity
Server end the handshake on one presented as a client certificate. So:

- CDD presents `server.crt` to the Identity Server only when it allows TLS
  client authentication, and warns at start when it does not. Set
  `tls.outbound_cert` and `tls.outbound_key` to present one anyway.
- The Identity Server presents its primary key pair only when it allows TLS
  client authentication, and logs that it does not. Give it a key pair of its
  own for CDD, as below.

`client_trust_store.pem` names who may call the sync endpoints. Put the
certificate the Identity Server presents in it, or a CA that issues only to the
Identity Server. With a CA that issues to anything else, also list the names
the Identity Server's certificate carries in `tls.client_cert_names`, so that
only it passes. Never put a public CA in it.

Turn `require_client_cert` off only where something in front of CDD ends the
TLS, such as a load balancer. The client certificate does not reach CDD there,
so require it at that load balancer, or at the service mesh, instead.

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

Replace it on both sides. A server certificate from a public CA does for
what browsers and SDKs reach; the certificates the Identity Server and CDD know
each other by come best from a CA of your own that issues only to them, or are
a self-signed pair, since public CAs no longer issue client certificates.

1. Here: `server.crt` and `server.key` with a certificate issued for the host
   in `server_url`, the name the Identity Server and the Console reach CDD on.
2. Here: `trust_store.pem` with the certificate the Identity Server presents,
   or the CA that issued it.
3. On the Identity Server: a key pair for CDD with the extended key usage
   `clientAuth`, in a keystore under `repository/resources/security`, set in
   its `deployment.toml`:

   ```toml
   [customer_data_service.client_certificate]
   file_name = "cdd-client.p12"
   type = "PKCS12"
   password = "..."
   alias = "cdd-client"
   ```

   `key_password` defaults to `password`.
4. Here: replace `client_trust_store.pem` with that certificate, or with the
   CA that issued it, and set `tls.client_cert_names` if that CA issues to
   anything else. Replace the file rather than adding to it: CDD refuses to
   start while it still trusts the default certificate beside certificates of
   this deployment, or while it presents a certificate of its own.
5. On the Identity Server: import the new `server.crt`, or its CA, into
   `client-truststore.p12`, and restart it.

In a container, mount the files rather than changing the image: this directory
for CDD, and the keystore and truststore for the Identity Server.
