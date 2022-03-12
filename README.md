# echo-streaming-bug

## Reproduce
To reproduce the issue:
1. Setup 2 WireGuard peers (see [quickstart](https://www.wireguard.com/quickstart/)) - your dev machine and some other machine (should be Linux)
2. Build the 2 binaries ([server](./cmd/server/) and [proxy](./cmd/proxy/))
```bash
GO111MODULE=on GOARCH=arm GOOS=linux go build -o server ./cmd/server/.
GO111MODULE=on GOARCH=arm GOOS=linux go build -o proxy ./cmd/proxy/.
```
3. Copy the 2 binaries and [certs](./certs/) on the other peer
4. SSH into the peer and proxy the 9000 port
```bash
ssh -L 9000:127.0.0.1:9000 <peer>
```
5. Start the server:
```bash
./server -cert cert.pem -key key.pem
```
6. Start the proxy:
```bash
./proxy -cert cert.pem -key key.pem
```
7.  Open https://localhost:9000 and run the following in the debugger console:
```js
async function streamData(url) {
    const res = await fetch(url);

    if (!res.ok) {
        throw `got response w/ status code: ${res.status}`;
    }
    
    const reader = res.body.getReader();

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
                console.error(e);
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
8. Now wait until you see:
```
SyntaxError: Unexpected token { in JSON at position 119
    at streamData (<anonymous>:20:40)
```

## Guides
Run the server:
```bash
go run ./cmd/server/main.go -cert ./certs/cert.pem -key ./certs/key.pem
```

Run the proxy:
```bash
go run ./cmd/proxy/main.go -cert ./certs/cert.pem -key ./certs/key.pem
```

Generate a self-signed SSL cert/key pair with (see [SO](https://stackoverflow.com/a/10176685/1092007)):
```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 365 -nodes -subj '/CN=localhost'
```
