// Command tfguard is the CLI entrypoint (Cobra).
package main

import "os"

func main() {
	os.Exit(execute(os.Stdout, os.Stderr, os.Args[1:]))
}
