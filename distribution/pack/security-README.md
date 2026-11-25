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

## The sync endpoints

The Identity Server sends CDD its users, profile schema, organizations,
service credentials and configuration on the sync endpoints, under
`/t/{org}/cds/api/v1/`:

```
POST  profiles/sync
POST  profile-schema/sync
POST  organizations/event-sync
POST  service-credentials/sync
PATCH config                     with Basic auth; the Console updates it with a token
```

They authenticate the Identity Server with the sync credentials: Basic auth
with `auth_server.admin_username` and `admin_password`. They are served on the
same listener as the APIs browsers and SDKs call, so they are reachable wherever
CDD is. In production:

1. Keep them off the public route. Only the Identity Server sends Basic auth to
   CDD, and the APIs applications call take tokens, so a load balancer or
   ingress in front of CDD for applications can refuse every request with an
   `Authorization: Basic` header. The Identity Server then reaches CDD on a
   private address.
2. Set a generated sync password, never the example one, on both sides:
   `AUTH_SERVER_ADMIN_PASSWORD` here and `admin_password` in the
   `[customer_data_service]` block of the Identity Server.
3. Where a leaked password alone must not be enough, require a client
   certificate as well (below).

## Requiring a client certificate

With `tls.require_client_cert: true`, the sync endpoints also refuse a request
without a client certificate that `client_trust_store.pem` verifies. The
listener asks every connection for one but requires none, so the APIs browsers
and SDKs call keep working. The Identity Server's CDD connector presents the
key pair of `[customer_data_service.client_certificate]` in its
`deployment.toml`, or of its primary keystore when that is not set; a connector
too old to present one gets 401 on every sync. It is off in the pack.

`client_trust_store.pem` names who may call the sync endpoints. Put the
certificate the Identity Server presents in it, or a CA that issues only to the
Identity Server. With a CA that issues to anything else, also list the names
the Identity Server's certificate carries in `tls.client_cert_names`, so that
only it passes. Never put a public CA in it.

A client certificate needs the TLS client authentication usage (extended key
usage `clientAuth`), or no extended key usage at all. Certificates from public
CAs are now for TLS server authentication only, and CDD ends the handshake on
one presented as a client certificate. So the Identity Server needs a key pair
of its own for CDD, from a CA of your own or self-signed:

```toml
[customer_data_service.client_certificate]
file_name = "cdd-client.p12"   # under repository/resources/security
type = "PKCS12"
password = "..."
alias = "cdd-client"           # key_password defaults to password
```

The connector presents its primary key pair only when it allows TLS client
authentication, and logs that it does not.

Where a load balancer in front of CDD ends the TLS, the client certificate does
not reach CDD: require it at that load balancer, or at the service mesh,
instead.

## Calls to the Identity Server

CDD calls the Identity Server's APIs (token, introspection, applications,
claims, SCIM) and authenticates with OAuth client credentials, like any other
client. With `tls.mtls_enabled: true` it also presents a client certificate,
which the Identity Server does not require: `tls.outbound_cert` and
`tls.outbound_key` when they are set, `server.crt` otherwise, and `server.crt`
only when it allows TLS client authentication. CDD warns at start when it does
not.

## What this pack ships

All four files are **Identity Server 7.3.0's default key pair**: the
`wso2carbon` entry of its `wso2carbon.p12`. A default Identity Server 7.3.0
presents that key pair and already trusts its certificate in its
`client-truststore.p12`. CDD presents it and trusts it here, so the two call
each other over TLS with no certificate set up on either side, and over mutual
TLS once `tls.require_client_cert` is on.

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
3. On the Identity Server: import the new `server.crt`, or its CA, into
   `client-truststore.p12`, and restart it.
4. With `tls.require_client_cert` on: give the Identity Server a client key
   pair of its own, as above, and replace `client_trust_store.pem` with its
   certificate, or the CA that issued it. Replace the file rather than adding
   to it: CDD refuses to start while it still trusts the default certificate
   beside certificates of this deployment, or while it presents a certificate
   of its own.

In a container, mount the files rather than changing the image: this directory
for CDD, and the keystore and truststore for the Identity Server.
