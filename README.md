# echo-streaming-bug
There seems to be some packet loss when using streaming responses in [echo](https://echo.labstack.com/), and when proxied through another echo server using the [proxy middleware](https://echo.labstack.com/middleware/proxy/).

I have tested in the following envs/contexts:
1. Run the server and proxy on a macOS 11.6.2 M1, with Go 1.16.4 and the client is Chrome 98.0/Firefox 97.0/Edge 99.0/Safari 15.2 (on the same macOS)
2. Run the server and proxy (cross-compiled for Linux, ARM, w/ Go 1.16.4) on a Raspberry Pi 3 (CM3+), Debian 9, Linux 4.14.98 and the client is the same as above (SSH and proxy - with `ssh -L 9000:127.0.0.1:9000 <host IP>` - the proxy server port to the local macOS machine)
3. Run same as 1, but multiple clients (Go CLI + browsers) connected

Notes:
1. When running everything on the same host, it takes much longer to get the error
2. When running the proxy and server remotely, it fails earlier, but still takes quite some time
3. Firefox fails much earlier (7 minutes on average) - it also seems like whenever I clear the console a few times, it makes the request fail with, but with a different error

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

In Firefox, when clearing the console, you get:
```
SyntaxError: JSON.parse: unexpected non-whitespace character after JSON data at line 2 column 1 of the JSON data
```

**NOTE**: I was unable to reproduce the error when running the Go [client](./cmd/client/). I ran it for about 2hrs and the error didn't occur.

## Guides
Generate a self-signed SSL cert/key pair with (see [SO](https://stackoverflow.com/a/10176685/1092007)):
```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 365 -nodes -subj '/CN=localhost'
```
