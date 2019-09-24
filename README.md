[![Build Status](https://travis-ci.org/dtgorski/midas.svg?branch=master)](https://travis-ci.org/dtgorski/midas)
[![Coverage Status](https://coveralls.io/repos/github/dtgorski/midas/badge.svg?branch=master)](https://coveralls.io/github/dtgorski/midas?branch=master)

## midas

Fast and cheap HTTP access logger middleware for Go.

### Installation
```
go get -u github.com/dtgorski/midas
```

### midas.Logger
... is a fast HTTP middleware for machine-readable response logging (_access.log_).
Special care has been taken in order to ensure marginal heap memory footprint and low garbage collector pressure.
The customized output layout has been chosen due to a specific requirement. If you need a different layout, feel free to fork.

#### Usage with a common router software:
```
import "github.com/dtgorski/midas"
.
.
router.Use(
    midas.Logger(os.Stdout),
    .
    .
)
```

#### Usage with handcrafted routing:
```
package main

import (
    "io"
    "log"
    "net/http"
    "os"

    "github.com/dtgorski/midas"
)

func main() {
    handler := http.HandlerFunc(
        func(w http.ResponseWriter, req *http.Request) {
            io.WriteString(w, "Hello, world!\n")
        },
    )
    logger := midas.Logger(os.Stdout)
    http.Handle("/", logger(handler))

    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

#### Log output could look like:
```
0000-00-00T00:00:00+00:00 | 127.0.0.1 | - | GET /hello HTTP/1.1 | 200 | - | - | Mozilla/5.0 (X11; Linux… | - | - | 14 | 0.000 | -
0000-00-00T00:00:00+00:00 | 127.0.0.1 | - | GET / HTTP/1.1 | 200 | - | - | Mozilla/5.0 (X11; Linux… | - | - | 14 | 0.000 | -
0000-00-00T00:00:00+00:00 | 127.0.0.1 | - | GET / HTTP/1.1 | 200 | - | - | Mozilla/5.0 (X11; Linux… | - | - | 14 | 0.000 | -
```
A log line consists of fields separated by a pipe and is suffixed by a newline (```\n```). Non-printable whitespace characters and UTF-8 runes will be replaced by a period sign (```.```). The fields in their order:
* access time 
* remote address without port
* remote user, if any
* request method, path, protocol
* response status code
* referer, if any
* forwarded for, if any
* user agent, if any
* SSL protocol, if any
* SSL cipher, if any
* bytes sent in response
* request time in seconds, three fractional digits 
* request id, if any

Although targeted for machine-reading (like log aggregators), the field values are padded with spaces for better human perception. 

#### Artificial benchmark:
```
$ make bench 
CGO_ENABLED=0 go test -run=^$ -bench=. -benchmem
goos: linux
goarch: amd64
pkg: github.com/dtgorski/midas

BenchmarkLoggerConcatFullLine-8   2000000   850 ns/op   48 B/op   2 allocs/op
```

## License
[MIT](https://opensource.org/licenses/MIT) - © dtg [at] lengo [dot] org
