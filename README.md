# echo-streaming-bug
There seems to be some packet loss when using streaming responses in [echo](https://echo.labstack.com/), and when proxied through another echo server using the [proxy middleware](https://echo.labstack.com/middleware/proxy/).

I have tested in the following envs/contexts:
1. Run the server, proxy and client on a macOS 11.6.2 M1, Go 1.16.4 and Chrome 98.0
2. Run the server and proxy (cross-compiled for Linux w/ Go 1.16.4) on a Raspberry Pi 3 (CM3+), Debian 9, Linux 4.14.98 and the client on a macOS 11.6.2 M1 and Chrome 98.0 (ssh and proxy the proxy port to the local macOS machine)
3. Run same as 1, but multiple clients (Go + browser) connected

Notes:
1. When running everything on the same host, it takes much longer to get the error
2. When running the proxy and server remotely, it fails earlier, but still takes quite some time

## Reproduce
To reproduce the issue:
1. Run the server:
```bash
go run ./cmd/server/main.go -cert ./certs/cert.pem -key ./certs/key.pem
```
2. Run the proxy:
```bash
go run ./cmd/proxy/main.go -cert ./certs/cert.pem -key ./certs/key.pem
```
3. Open https://localhost:9000 and run the following in the debugger console:
```js
async function streamData(url) {
    const res = await fetch(url);

    if (!res.ok) {
        throw `got response w/ status code: ${res.status}`;
    }
    
    const reader = res.body.getReader();
    const t0 = window.performance.now();

    while (true) {
        try {
            const {done, value} = await reader.read();
            if (done) {
                console.info("done")
                return;
            }
    
            try {
                const res = new Response(value);
                const data = await res.json();
                console.log(data);
            } catch (e) {
                const t1 = window.performance.now();
                console.error(e);
                console.info(`${(t1 - t0)/1000}s`);
                return;
            }
        } catch (e) {
            console.error(e);
            return;
        }
    }
}

streamData("https://localhost:9000/api/ping?interval=100ms")
```

You should see (in the browser debugger) - after some time (30 mins or more):
```
SyntaxError: Unexpected token { in JSON at position 119
    at streamData (<anonymous>:20:40)
```

Note: I was unable to reproduce the error when running the Go [client](./cmd/client/). I ran it for about 2hrs and the error didn't occur.

## Guides
Generate a self-signed SSL cert/key pair with (see [SO](https://stackoverflow.com/a/10176685/1092007)):
```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 365 -nodes -subj '/CN=localhost'
```
