package main

import (
    "fmt"
    "noarch"
)

func main() {
    var buffer []byte
    noarch.Printf([]byte("%s (%d b) == %s (%d b) # got \"%s\"\x00"),
        []byte("buffer\x00"),
        noarch.Strlen([]byte("buffer\x00")),
        []byte("\"3 a 1.999 42.500 \"\x00"),
        noarch.Strlen([]byte("\"3 a 1.999 42.500 \"\x00")),
        buffer,
    )
}