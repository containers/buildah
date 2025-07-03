package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <password>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate a password hash using golang.org/x/crypto/bcrypt.\n")
		os.Exit(1)
	}

	passwd, err := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating password hash: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(passwd))
}
