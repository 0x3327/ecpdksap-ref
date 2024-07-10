package main

import (
	"ecpdksap-bn254/recipient"
	"ecpdksap-bn254/sender"
	"fmt"
	"os"
)

func main () {

	if len(os.Args) != 3 {
		fmt.Println("\nERR: All subcommands (first arg.) receive only one param (second arg.) .\n")
		return
	}

	subcmd := os.Args[1];
	arg := os.Args[2];

	switch subcmd {

		case "send":
		
			r, _, _, _ := sender.Send(arg)
			fmt.Println(r)
			
		case "receive-scan":

			rP := recipient.Scan(arg)
			fmt.Println(rP)

		case "receive-scan-using-vtag":
			rP := recipient.ScanUsingViewTag(arg)
			fmt.Println(rP)
			

		default:
			fmt.Printf("\nERR: only: `send` | `receive-scan` | `receive-scan-using-vtag` subcommands allowed.\n\n")
			return
	}
}