# GoOSC

[Open Sound Control (OSC)](http://opensoundcontrol.org/introduction-osc)
library for Golang. Implemented in pure Go.

* Build Status:  [![CI Status][CIStatus]][CIProject]
* Documentation: [![GoDoc][GoDocStatus]][GoDoc]

## Features
  * OSC Bundles, including timetags
  * OSC Messages
  * OSC Client
  * OSC Server
  * Supports the following OSC argument types:
    * 'i' (Int32)
    * 'f' (Float32)
    * 's' (string)
    * 'b' (blob / binary data)
    * 'h' (Int64)
    * 't' (OSC timetag)
    * 'd' (Double/int64)
    * 'T' (True)
    * 'F' (False)
    * 'N' (Nil)
  * Support for OSC address pattern including '\*', '?', '{,}' and '[]' wildcards

## Usage

### Client

```go
import osc "github.com/kward/go-osc"

func main() {
    client := osc.NewClient("localhost", 8765)
    msg := osc.NewMessage("/osc/address")
    msg.Append(int32(111))
    msg.Append(true)
    msg.Append("hello")
    client.Send(msg)
}
```

### Server

```go
package main

import "github.com/kward/go-osc/osc"

func main() {
  addr := "127.0.0.1:8765"
  server, err := osc.NewServer(addr)
  if err != nil {
    panic(err)
  }

  server.Handle("/message/address", func(msg *osc.Message) {
    println(msg.String())
  })

  server.ListenAndServe()
}
```

## Misc
This library was forked from https://github.com/hypebeast/go-osc to modernize the codebase with Go modules, GitHub Actions CI/CD, and updated dependencies.


<!--- Links -->

[CIProject]: https://github.com/kward/go-osc/actions
[CIStatus]: https://github.com/kward/go-osc/workflows/CI/badge.svg

[GoDoc]: https://godoc.org/github.com/kward/go-osc/osc
[GoDocStatus]: https://godoc.org/github.com/kward/go-osc/osc?status.svg
