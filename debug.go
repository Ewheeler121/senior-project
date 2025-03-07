//go:build !release
package main

import "fmt"

func debugPrint(v ...interface{}) {
    fmt.Println(v...)
}
