tlsutil
========

Building Certificates
=====================

The testcerts/buildcerts script generates pairs of self-signed certs,
and sets them up to trust each other explicitly.

Setting up 2way HTTPS Server
===================

There are working, compiling examples in the tests 
in this directory.  `github.com/deciphernow/encryptedfs` uses this to 
create its 2way ssl listener:

```go
	// Create an http handler out of all of it wired together
	handler := handlers.NewRestHandler(fsf, authm, logger)

	serverTrust := "certs/server.trust.pem"
	serverCert := "certs/cert.pem"
	serverKey := "certs/private_key.pem"

	cfg, err := tlsutil.BuildServerTLSConfig(
		serverTrust,
		serverCert,
		serverKey,
	)
	if err != nil {
		logger("! Test unable to get server config: %v", err)
		return
	}
	httpServer := &http.Server{
		Addr:      fmt.Sprintf("%s:%d", handlers.BindAddress, port),
		Handler:   handler,
		TLSConfig: &cfg,
	}
	httpServer.ListenAndServeTLS(serverCert, serverKey)
```

The older package `tlsutil` should not be used, as it does not check
certificates properly, and exposes too much detail to bother reaching
into another library.  The commons should be setting policy so that
common functionality behaves commonly across products.

Setting up 2way HTTPS client
============================

```go
		// client parameters are filenames of pem files
		// serverCN is the CN we expect in the server we
		// connect to.  if it isn't the DNS name,
		// then you will need to get serverCN from an
		// environment variable that knows how that cert
		// is issued.
		clientConnFactory, err := tlsutil.NewTLSClientConnFactory(
			clientTrust,
			clientCert,
			clientKey,
			serverCN,
			host,
			fmt.Sprintf("%d", port),
		)

		... //error checks

		res, err := clientConnFactory.Do(req)

		... // error checks
		if res.StatusCode != http.StatusOK {
			...
		}

```

TLS Server TCP
==============

To create a TLS tcp socket without an application level protocol, you must create the server socket and sit in loops doing an accept on the socket:

```go
	cfg, err := tlsutil.BuildServerTLSConfig(serverTrust, serverCert, serverKey)
	... // error checks

	host := "127.0.0.1"
	addr := fmt.Sprintf("%s:%d", host, port)
	serverSock, err := tls.Listen("tcp", addr, &cfg)
	
	...

	for {
		...
		// conn.Read, conn.Write, conn.Close
		conn, err := serverSock.Accept()
		...
	}
```

In this example, `conn` is an io.ReadCloser, and io.Writer.  You can read
and write bidirectionally to a client TLS connection.

TLS Client TCP
==============

Again, in addition to knowing the host and port to connect to,
we must also know the serverCN.  Do NOT InsecureSkipVerify,
because it is not just disabling hostname checks, but ALL certificate checks.
There isn't a way to turn it on in production when
InsecureSkipVerify is used because TLS code must have some way of 
getting the serverCN, either by setting it to "" and requiring 
that the certificate CN and hostname match, or knowing the CN 
that will be in the server (common in clusters that share the same certificate).

```go
		// clientSocket.Read, clientSock.Write, clientSock.Close
		clientSock, err := tlsutil.NewTLSClientConn(
			clientTrust, clientCert, clientKey,
			serverCN,
			host,
			fmt.Sprintf("%d", port),
		)

```

