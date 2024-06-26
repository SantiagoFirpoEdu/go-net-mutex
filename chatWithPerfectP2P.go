// Construido como parte da disciplina: Sistemas Distribuidos - PUCRS - Escola Politecnica
//  Professor: Fernando Dotti  (https://fldotti.github.io/)

/*
LANCAR 2 PROCESSOS EM SHELL's DIFERENTES, PARA CADA PROCESSO, O SEU PROPRIO ENDERECO EE O PRIMEIRO DA LISTA
go run chatComPPLink.go   127.0.0.1:5001  127.0.0.1:6001
go run chatComPPLink.go   127.0.0.1:6001  127.0.0.1:5001
ou, claro, fazer distribuido trocando os ip's
*/

package main

import (
	"SD/PerfectP2PLink"
	"bufio"
	"fmt"
	"os"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage:   go run chatComPPLink.go thisProcessIpAddress:port otherProcessIpAddress:port")
		fmt.Println("Example: go run chatWithPerfectP2P.go 127.0.0.1:8050 127.0.0.1:8051")
		fmt.Println("Example: go run chatWithPerfectP2P.go 127.0.0.1:8051 127.0.0.1:8050")
		return
	}

	addressArgs := os.Args[1:]
	addresses := make([]PerfectP2PLink.Address, len(addressArgs))
	for i, address := range addressArgs {
		addresses[i] = PerfectP2PLink.Address(address)
	}
	fmt.Println("Chat PPLink - addresses: ", addresses)

	perfectP2PLink := PerfectP2PLink.NewLink(addresses[0], true)

	go func() {
		for {
			m := <-perfectP2PLink.ReceiveChannel()
			fmt.Println("Rcv: ", m)
		}
	}()

	go func() {
		for {
			fmt.Print("Snd: ")
			scanner := bufio.NewScanner(os.Stdin)
			var msg string

			if scanner.Scan() {
				msg = scanner.Text()
			}
			req := PerfectP2PLink.SentRequest{
				To:      addresses[1],
				Message: msg}

			perfectP2PLink.SendChannel() <- req
		}
	}()

	<-(make(chan int))
}
