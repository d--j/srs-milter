# SRS milter

![Build status](https://github.com/d--j/srs-milter/actions/workflows/go.yml/badge.svg?branch=main)
[![codecov](https://codecov.io/gh/d--j/srs-milter/branch/main/graph/badge.svg?token=5R5EVF5VEO)](https://codecov.io/gh/d--j/srs-milter)

A milter (mail filter) for Postfix and Sendmail
handling [SRS address rewriting](https://en.wikipedia.org/wiki/Sender_Rewriting_Scheme).

## Features

* Compatible with Postfix and Sendmail
* Lazy SRS rewriting: only rewrite when email is not local and the SPF record of the destination prevents us from
  sending email for it
* Reverse Rewriting: Rewrite RCPT TO and the To-Header (only when message is not DKIM signed)
* Reload configuration from configuration file automatically when the file changes
* Support for secret key rollover
* Fully IDNA-compatible
* Automatic integration test suite testing the interoperability with Postfix and Sendmail

## Installation

Download the [latest release](https://github.com/d--j/srs-milter/releases/latest) for your OS and architecture.

We have packages for Debian/Ubuntu, Alpine and RPM based linux distributions.
For FreeBSD/NetBSD/OpenBSD or macOS we only have pre-compiled executables.
You need to set up the daemon and configuration on your own on those platforms.

## Configuration

`srs-milter` will search its configuration in `/etc/srs-milter/srs-milter.yml` or `./srs-milter.yml`.

You need to set these two configuration options in it (the packages will try to do that for you):

```yaml
# Required: The domain we use to generate SRS addresses
srsDomain: 'srs.example.com'

# Required: List of secrets for SRS address generation and validation
# The first key gets used for SRS address generation. All other keys
# are only used for SRS address validation.
# You can use multiple keys for key rotation.
srsKeys:
  - active-key
  - rotated-key
```

It is highly encouraged to also set the list of local domains. If you do not do this, we will consider all destinations
to be external. When you properly set up SPF for all your domains, we will not SRS rewrite local domains. But you can
prevent unnecessary DNS lookups when you define the list of local domains:

```yaml
# All domains we consider local (i.e. we do not forward but deliver locally)
# You can use IDN domain names. They will be normalized to their ASCII representation automatically.
localDomains:
  - 'example.net'
  - 'example.com'
```

You can also specify an optional MySQL query for email forwarding lookups: 

```yaml
# Optional: You can specify a MySQL connection/query to lookup mail forwarding replacements
dbDriver: 'mysql'
dbDSN: 'user:password@tcp(host:port)/dbname'
dbForwardQuery: "SELECT destination from mail_forwarding WHERE source = ? AND active = 'y' AND server_id = 1;"
```

If your machine does not have public IP addresses (NATed/firewalled) or you deployed the milter on another machine, you
need to specify the IPs that we check against the SPF records. These IPs should be the IPs that get used for outgoing
SMTP connections.

```yaml
# Public IPv4 and IPv6 addresses of the MTA
# If the MTA is on another host or does not have public IPs (e.g. it is firewalled) you need
# to specify this list.
# If you leave this list empty we will try to determine the public IPs automatically.
localIps:
  - '8.8.8.8'
```

`srs-milter` will listen on `127.0.0.1` port `10382` for milter requests.
You can use command line parameters to change this default:

```
$ srs-milter -help
Usage of ./srs-milter:
  -milterAddr address/port
        Bind milter server to address/port or unix domain socket path (default "127.0.0.1:10382")
  -milterProto family
        Protocol family (unix or tcp) of milter server (default "tcp")
  -socketmapAddr address/port
        Bind socketmap server to address/port or unix domain socket path (default "127.0.0.1:10383")
  -socketmapProto family
        Protocol family (unix or tcp) of socketmap server (default "tcp")
  -forward email
        email to do forward SRS lookup for. If specified the milter will not be started.
  -reverse email
        email to do reverse SRS lookup for. If specified the milter will not be started.
  -systemd
        enable systemd mode (log without date/time)
```

## MTA configuration

### Postfix

Add this to `/etc/postfix/main.cf`

```
smtpd_milters = inet:127.0.0.1:10382
non_smtpd_milters = inet:127.0.0.1:10382
milter_protocol = 6
# if you use a dedicated SRS domain (what you should do) then you need to tell Postfix to accept SRS bounces to this domain.
# You could do that with e.g.:
relay_domains = hash:your/relay/domain/list srs.example.com
recipient_canonical_maps = socketmap:inet:localhost:10383:decode
recipient_canonical_classes = envelope_recipient
```

If you already have milters defined (e.g. Rspamd),
add the `srs-milter` entry to the beginning of the `smtp_milters`/`non_smtpd_milters` list.
It works at any place but the other milters might benefit from `srs-milter` to run first.

### Sendmail

Add this to your `sendmail.mc`:

```
INPUT_MAIL_FILTER(`srsmilter', `S=inet:10382@localhost')
```

And call `sendmailconfig` or `make` (or your way of generating `sendmail.cf` from the M4 macros).

If you fiddle with `sendmail.cf` directly you probably know what to do to activate a milter (
add `Xsrsmilter, S=inet:10382@localhost` and then add `srsmilter` to the `InputMailFilters` list of `DaemonPortOptions`)

## Caveats

We assume that you set up SPF correctly – if the MTA has multiple IP addresses and any one of them is allowed to send
for a domain, we assume that we are allowed to send emails for this domain. If not all the public IPs of the MTA are
allowed to send, and you misconfigured your MTA to pick a disallowed IP this will result in excessive SRS rewriting.

The SPF lookups are cached for 30 minutes.
We do support SPF macros, but we cache the SPF result without taking the source address or sender into account.
If you have complicated dynamic SPF rules we might cache things too aggressively. This should not be a big problem if
you manually specified the local domains.

Currently, we only support embedded SRS rewriting, and you cannot configure the hash length or the SRS delimiter.

## License

BSD 2-Clause
