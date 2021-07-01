package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func getExitSignalsChannel() chan os.Signal {

	c := make(chan os.Signal, 1)
	signal.Notify(c,
		// https://www.gnu.org/software/libc/manual/html_node/Termination-Signals.html
		syscall.SIGTERM, // "the normal way to politely ask a program to terminate"
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGQUIT, // Ctrl-\
		// syscall.SIGKILL, // "always fatal", "SIGKILL and SIGSTOP may not be caught by a program"
		// syscall.SIGHUP, // "terminal is disconnected"
	)
	return c

}

func (transport *Transport) Exit() {
	<-transport.exitChan
	transport.fileDestination.Close()
	transport.conn.Close()
	log.Println("Shutting down")
	os.Exit(0)

}

// func getNewLogSignalsChannel() chan os.Signal {

// 	c := make(chan os.Signal, 1)
// 	signal.Notify(c,
// 		// https://www.gnu.org/software/libc/manual/html_node/Termination-Signals.html
// 		syscall.SIGHUP, // "terminal is disconnected"
// 	)
// 	return c

// }

// newLogChan := getNewLogSignalsChannel()

// go func() {
// 	<-newLogChan
// 	log.Println("Received a signal from logrotate, close the file.")
// 	writer.Flush()
// 	filetDestination.Close()
// 	log.Println("Opening a new file.")
// 	time.Sleep(2 * time.Second)
// 	writer = openOutputDevice(cfg.NameFileToLog)

// }()
