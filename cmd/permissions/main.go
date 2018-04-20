package main

import (
	"fmt"
	"os"
)

func exitOnError(err error) {
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func main() {
	exitOnError(setupLogger())

	_, err := setupDB()
	exitOnError(err)
}
