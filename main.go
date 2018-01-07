package main

import (
    "github.com/jessevdk/go-flags"
    "os"
    "golang.org/x/sys/windows/svc"
    "log"
)

type Options struct {
    Verbose           bool   `short:"v" long:"verbose"             description:"Verbose mode"`
}

var Opts Options
var optsParser = flags.NewParser(&Opts, flags.Default)


// test doc123
func main() {
    isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}

	if !isIntSess {
        runService("zohm", false)
        return
    }

    if _, err := optsParser.Parse(); err != nil {
        if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
            os.Exit(0)
        } else {
            os.Exit(1)
        }
    }
}
