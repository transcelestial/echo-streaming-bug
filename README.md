# echo-streaming-bug
There seems to be some packet loss when using streaming responses in [echo](https://echo.labstack.com/), and when proxied through another echo server using the [proxy middleware](https://echo.labstack.com/middleware/proxy/).

I have tested in the following envs/contexts:
1. Run the server and proxy on a macOS 11.6.2 M1, with Go 1.16.4 and the client is Chrome 98.0/Firefox 97.0/Edge 99.0/Safari 15.2 (on the same macOS)
2. Run the server and proxy (cross-compiled for Linux, ARM, w/ Go 1.16.4) on a Raspberry Pi 3 (CM3+), Debian 9, Linux 4.14.98 and the client is the same as above (SSH and proxy - with `ssh -L 9000:127.0.0.1:9000 <host IP>` - the proxy server port to the local macOS machine)
3. Run same as 1, but multiple clients (Go CLI + browsers) connected
4. Run same as 1, but no proxy, so just the server + the clients
5. Run same as 1, but non-echo server and proxy

Issue reported at:
1. https://github.com/labstack/echo/issues/2125
2. https://github.com/golang/go/issues/51646
3. https://bugs.chromium.org/p/chromium/issues/detail?id=1305928

## Reproduce

### Echo + Echo Proxy
To reproduce the issue on your local machine using the echo server and proxy:

1. Run the echo server:
```bash
go run ./cmd/echoserver/main.go -cert ./certs/cert.pem -key ./certs/key.pem
```
2. Run the echo proxy:
```bash
go run ./cmd/echoproxy/main.go -cert ./certs/cert.pem -key ./certs/key.pem
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
                console.info("done");
                reader.cancel();
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
                reader.cancel();
                return;
            }
        } catch (e) {
            console.error(e);
            reader.cancel();
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

### Echo + HTTP Proxy:
To reproduce the issue on your local machine using the echo server and the `net/http/httputil` proxy:

1. Run the echo server:
```bash
go run ./cmd/echoserver/main.go -cert ./certs/cert.pem -key ./certs/key.pem
```
2. Run the HTTP proxy:
```bash
go run ./cmd/httpproxy/main.go -cert ./certs/cert.pem -key ./certs/key.pem
```
3. Run the same code as in the [Echo + Echo Proxy](#echo-+-echo-proxy) example, step 3, but different URL (so you just need the `streamData` func):
```js
streamData("https://localhost:9000/ping?interval=100ms")
```

### HTTP Server + HTTP Proxy:
To reproduce the issue on your local machine using the `net/http` server and the `net/http/httputil` proxy:

1. Run the HTTP server:
```bash
go run ./cmd/httpserver/main.go -cert ./certs/cert.pem -key ./certs/key.pem
```
2. Run the HTTP proxy:
```bash
go run ./cmd/httpproxy/main.go -cert ./certs/cert.pem -key ./certs/key.pem
```
3. Run the same code as in the [Echo + HTTP Proxy](#echo-+-http-proxy) example, step 3:

### Echo + Echo Proxy in Docker:
To reproduce the issue on your local machine using the echo server and proxy with Docker:
1. Build the images:
```bash
docker-compose build
```
2. Run the services:
```bash
docker-compose up
```
3. Run the same code as in the [Echo + Echo Proxy](#echo-+-echo-proxy) example, step 3

## Symptoms
1. When running everything on the same host, it takes much longer to get the error
2. When running the proxy and server remotely, it fails earlier, but still takes quite some time
3. I was able to get the same failures w/o using the proxy, so using just the server(s) - either echo or non-echo
4. Chrome and Edge fail consistenly at ~ 1519s; this number seems to be lower when the JSON payload is higher
5. Firefox fails much earlier (15 minutes on average)
6. In Firefox, when clearing the console (a few times) or when leaving the browser window, you get:
```
SyntaxError: JSON.parse: unexpected non-whitespace character after JSON data at line 2 column 1 of the JSON data
```
7. Firefox fails fast if the JSON payload is a little larger (sometimes with the error described above):
```
SyntaxError: JSON.parse: unterminated string at line 1 column 445 of the JSON data
```
8. Chrome, Firefox and Edge fail when going to fullscreen then back to normal
9. I was unable to reproduce the error when running the Go [client](./cmd/client/). I ran it for about 2hrs and the error didn't occur.

### Req/Res Headers

#### Chrome
```
REQUEST ->

Accept: */*
Accept-Encoding: gzip, deflate, br
Accept-Language: en-US,en;q=0.9,da;q=0.8,ro;q=0.7,fr;q=0.6,de;q=0.5
Referer: https://localhost:9000/
Sec-Ch-Ua: " Not A;Brand";v="99", "Chromium";v="98", "Google Chrome";v="98"
Sec-Ch-Ua-Mobile: ?0
Sec-Ch-Ua-Platform: "macOS"
Sec-Fetch-Dest: empty
Sec-Fetch-Mode: cors
Sec-Fetch-Site: same-origin
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36

<- RESPONSE

Content-Encoding: gzip
Content-Type: application/json
Date: Sun, 13 Mar 2022 14:04:32 GMT
Vary: Origin
Vary: Accept-Encoding
```

#### Firefox
```
REQUEST ->

Accept: */*
Accept-Encoding: gzip, deflate, br
Accept-Language: en-US,en;q=0.5
Referer: https://localhost:9000/
Sec-Fetch-Dest: empty
Sec-Fetch-Mode: cors
Sec-Fetch-Site: same-origin
Te: trailers
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:97.0) Gecko/20100101 Firefox/97.0]

<- RESPONSE

Content-Encoding: gzip
Content-Type: application/json
Date: Mon, 14 Mar 2022 01:48:25 GMT
Vary: Origin
Vary: Accept-Encoding
X-Firefox-Spdy: h2
```

#### Safari
```
REQUEST ->

Accept: */*
Accept-Encoding: gzip, deflate, br
Accept-Language: en-sg
Referer: https://localhost:9000/
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.2 Safari/605.1.15

<- RESPONSE

Content-Encoding: gzip
Content-Type: application/json
Vary: Origin, Accept-Encoding
Date: Sun, 13 Mar 2022 14:05:30 GMT
```

#### Edge
```
REQUEST ->

Accept: */*
Accept-Encoding: gzip, deflate, br
Accept-Language: en-GB,en;q=0.9,en-US;q=0.8
Referer: https://localhost:9000/
Sec-Ch-Ua: " Not A;Brand";v="99", "Chromium";v="99", "Microsoft Edge";v="99"
Sec-Ch-Ua-Mobile: ?0
Sec-Ch-Ua-Platform: "macOS"
Sec-Fetch-Dest: empty
Sec-Fetch-Mode: cors
Sec-Fetch-Site: same-origin
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.51 Safari/537.36 Edg/99.0.1150.39

<- RESPONSE

Content-Encoding: gzip
Content-Type: application/json
Date: Sun, 13 Mar 2022 14:06:11 GMT
Vary: Origin
Vary: Accept-Encoding
```

#### Go client
```
REQUEST ->

Accept-Encoding: gzip
User-Agent: Go-http-client/1.1

<- RESPONSE

No headers provided
```

## Fix
After some investigation and help from [@seankhliao](https://github.com/seankhliao), the fix was as simple as making sure to only parse the data to JSON after line endings (because the stream can be chunked arbitrarily):
```js
streamData("https://localhost:9000/ping?interval=100ms")

async function streamData(url) {
    const res = await fetch(url);

    if (!res.ok) {
        throw `got response w/ status code: ${res.status}`;
    }
    
    const reader = res.body.getReader();
    const t0 = window.performance.now();

    try {
        for await (const line of readLineByLine(reader)) {
          const json = await toJSON(line)
          console.info('Got data', json);
        }
        
        const t1 = window.performance.now();
        const duration = (t1 - t0)/1000;

        console.info(`Stream ended after ${duration}s`);
    } catch (e) {
        const t1 = window.performance.now();
        const duration = (t1 - t0)/1000;
        if (e.name === 'AbortError') {
          console.info(`Stream aborted after ${duration}s`);
        } else {
            console.info(`Stream errored after ${duration}s`);
            console.error(e);
        }
    }

    reader.cancel();
}

async function* readLineByLine(reader) {
    const utf8Decoder = new TextDecoder("utf-8");
    let {value: chunk, done: readerDone} = await reader.read();
    chunk = chunk ? utf8Decoder.decode(chunk, {stream: true}) : "";

    const re = /\r\n|\n|\r/gm;
    let startIndex = 0;

    for (; ;) {
        const result = re.exec(chunk);
        if (!result) {
            if (readerDone) {
                break;
            }

            const remainder = chunk.substr(startIndex);

            ({value: chunk, done: readerDone} = await reader.read());
            chunk = remainder + (chunk ? utf8Decoder.decode(chunk, {stream: true}) : "");
            startIndex = re.lastIndex = 0;

            continue;
        }

        yield chunk.substring(startIndex, result.index);

        startIndex = re.lastIndex;
    }

    if (startIndex < chunk.length) {
        // last line didn't end in a newline char
        yield chunk.substr(startIndex);
    }
}

async function toJSON(value) {
    try {
        const res = new Response(value);
        const data = await res.json();
        return data;
    } catch (e) {
        throw e;
    }
}
```

## Guides
Generate a self-signed SSL cert/key pair with (see [SO](https://stackoverflow.com/a/10176685/1092007)):
```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 365 -nodes -subj '/CN=localhost'
```
