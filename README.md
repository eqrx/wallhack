![Go Reference](https://pkg.go.dev/badge/eqrx.net/wallhack.svg)

# wallhack

Connect two linux tun interfaces over a TLS connection together. Use systemd as hard as possible for that. I use this
as a tunnel for wireguard connections from and to nodes that are behind insane network setups.

This project is released under GNU Affero General Public License v3.0, see LICENCE file in this repo for more info.

## How it works

wallhack consists of a client and a server side. The client attaches to an existing tun device named `wallhack` and 
opens a TLS connection to a configured wallhack server instance. It authenticates with the server by a client cert. 
The server part validates incoming connections and attaches to an existing tun devices named like the common name of 
the client certificate. Frames arriving at one tun are copied 1:1 to the other tun and vice versa - no translation 
or rewriting of any kind is done.

### Advantages

- You can punch through restrictive firewalls that only allow https egress traffic (run-off-the-mill hotel wifi).
- wallhack does not care how you configure your tuns or what you send through them.

### Disadvantages

- wallhack encapsulates all network traffic into an encrypted TCP stream. This means it is not fast. If you want a 
  low latency, high bandwidth solution I would recommend looking for other solutions. I use this project to allow my 
  mobile devices to communicate with wireguard peers in my home network even when in a restricted network abroad. I 
  do not mind it being slow as long as I can access my files while on the other side of the country.

## How to set it up

### Packages

I maintain the AUR package for archlinux. It contains a sysusers config, a netdev file as well as the service 
units for both clients and servers and the wallhack binary.

### Create TLS certs

Clients need one client certificate each that has a unique common name (CN) set that identifies the client.
This name will be used to map clients to tun devices. Since tun devices may only have names with a maximum of 
15 bytes (not chars) you should not go over that. You can setup a client PKI infrastructure with easyrsa.

The server needs a certificate that is accepted by the clients system CA store. I would recommend using a letsencrypt
certificate for that. The server also gets the certificate of the client CA so it can validate them.

### Create system users

Wallhack needs to have a system user. Normally I like to use dynamic users but the tun devices need to to be assigned
to this UID to wallhack may interact with them. Either use the archlinux package or put the 
[configuration file](init/sysuser.conf) to `/etc/sysusers.d` and let it be 
[picked up](https://www.freedesktop.org/software/systemd/man/systemd-sysusers.html).

### Create tun devices

Tell systemd-networkd to create netdev devices for you on each client. To do this you install [this](init/wallhack.netdev) 
to `/etc/systemd/network/wallhack.netdev` (or use the AUR package ;)).

On the server side you need a tun device for each client. To do that copy the [same file](init/wallhack.netdev) 
to `/etc/systemd/network/<client name>.netdev` and change the line `Name=wallhack` in the `[Match]` section to
`Name=<client name>`.

### Create unit files

Copy the [server unit](init/server.service) to `/etc/systemd/system/wallhack-server.service` on the server side 
and the [client unit](init/client.service) `/etc/systemd/system/wallhack-client.service` on the client sides.

The server gets its listening socket passed by systemd. To configure that create the file 
`/etc/systemd/system/wallhack-server.socket` with the follwing content and see 
[here](https://www.freedesktop.org/software/systemd/man/systemd.socket.html) for more info:

```
[Socket]
# Let wallhack bind to this addr:port. You can this line multiple times to let wallhack listen on multiple ports.
ListenStream=1.2.3.4:443 
FreeBind=true # Allow listening on the given ports before the interface is up.

[Install]
WantedBy=sockets.target
```

Don't forget to enable and start the services after you are done configuring.

### Create credentials

Wallhack uses systemd credentials to access secrets..

The server needs the following content:
- Private TLS key in `/etc/wallhack/key`
- TLS Certificate chain to present to clients in `/etc/wallhack/cert`
- CA used to verify clients in `/etc/wallhack/ca`


The client needs:
- Private TLS key in `/etc/wallhack/key`
- TLS Certificate chain to present the server in `/etc/wallhack/cert`

To create a credential file, use something like this 
`systemd-creds encrypt <unencrypted cert file pat> /etc/wallhack/key`.

### Provide the wallhack binary

Run `build.sh` in the root of this project and put the resulting `bin/wallhack` at `/usr/bin/wallhack` onto 
all servers and clients.

### Do the actual network configuration

Wallhack would run just fine with the configuration so far but no traffic would run over it. This is because 
the tun devices have neither addresses nor routes assigned. What configuration you want to apply here soley depends
on your use case. Let's get you started with a basic example. We have a server and a single client named `chicken` 
and we want to let them communicate over the tunnel. To do so I generated myself a random IPv6 prefix: 
`fd0d:5619:c605:0::/64`. the server gets the address `fd0d:5619:c605:0::1/64`, the client `fd0d:5619:c605:0::2/64`.

Create this file at `/etc/systemd/network/chicken.network` on the server:

```
[Match]
Name=chicken

[Network]
Address=fd0d:5619:c605:0::1/64
```

Create this file at `/etc/systemd/network/wallhack.network` on the client:

```
[Match]
Name=wallhack

[Network]
Address=fd0d:5619:c605:0::2/64
```

Since both addresses are in the same subnet both sides automatically get routes attached. To start it all up 
run a `networkctl reload` on both sides, start `wallhack-server.socket` on the server and 
`wallhack-client.service` on the client.
