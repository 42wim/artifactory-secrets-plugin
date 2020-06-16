This is not yet ready for production use. Please file issues though as you find them.

![Build](https://github.com/idcmp/artifactory-secrets-plugin/workflows/Build/badge.svg)

# Vault Artifactory Secrets Plugin

This is a [HashiCorp Vault](https://www.vaultproject.io/) plugin which talks to JFrog Artifactory server (5.0.0 or later) and will
dynamically provision access tokens with specified scopes. This backend can be mounted multiple times
to provide access to multiple Artifactory servers.

Using this plugin, you limit the accidental exposure window of Artifactory tokens; useful for continuous
integration servers.

## Testing Locally

If you're compiling this yourself and want to do a local sanity test, you
can do something like:

```bash
terminal-1$ make
...

terminal-2$ export VAULT_ADDR=http://127.0.0.1:8200
terminal-2$ export VAULT_TOKEN=root
terminal-2$ make setup
...

terminal-2$ make artifactory &  # Runs netcat returning a static JSON response
terminal-2$ vault read artifactory/token/test
```


## Usage

To actually integrate it into Vault:

```bash
$ vault secrets enable artifactory

# Also supports max_ttl= and default_ttl=
$ vault write artifactory/config/admin \
               url=https://artifactory.example.org/artifactory \
               access_token=0ab31978246345871028973fbcdeabcfadecbadef

# Also supports grant_type=, and audience= (see JFrog documentation)
$ vault write artifactory/roles/jenkins \
               username="example-service-jenkins" \
               scope="api:* member-of-groups:ci-server" \
               default_ttl=1h max_ttl=3h 

$ vault list artifactory/roles
Keys
----
jenkins

$ vault read artifactory/token/jenkins 
Key                Value
---                -----
lease_id           artifactory/token/jenkins/25jYH8DjUU548323zPWiSakh
access_token       adsdgbtybbeeyh...
role               jenkins
scope              api:* member-of-groups:ci-server
```

## Access Token Creation

This backed creates access tokens in Artifactory whose expiry is the "max_ttl" of
either the role or the backend. If the lease is revoked before "max_ttl", then Vault asks
Artifactory to revoke the token in question. 

If the "max_ttl" is 0, then the access token will be created without an expiry, and Vault
will revoke it when the owning token expires.

Do you wish the access tokens could be scoped to a specific network block (like only your
CI network)? Vote on [RTFACT-22477](https://www.jfrog.com/jira/browse/RTFACT-22477) on JFrog's Jira.
