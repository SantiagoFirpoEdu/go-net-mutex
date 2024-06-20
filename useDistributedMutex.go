// Construido como parte da disciplina: Sistemas Distribuidos - PUCRS - Escola Politecnica
//  Professor: Fernando Dotti  (https://fldotti.github.io/)
/* fmt.Println("go run useDistributedMutex.go 0 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
   fmt.Println("go run useDistributedMutex.go 1 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
   fmt.Println("go run useDistributedMutex.go 2 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
/*
  LANCAR N PROCESSOS EM SHELL's DIFERENTES, UMA PARA CADA PROCESSO.
  para cada processo: seu id único e a mesma lista de processos.
  o endereco de cada processo é o dado na lista, na posicao do seu id.
  no exemplo acima o processo com id=1  usa a porta 6001 para receber e as portas
  5000 e 7002 para mandar mensagens respectivamente para processos com id=0 e 2
*/

package main

import (
	"SD/DistributedMutex"
	"SD/PerfectP2PLink"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Please specify at least one address:port!")
		fmt.Println("go run useDistributedMutex.go 0 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
		fmt.Println("go run useDistributedMutex.go 1 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
		fmt.Println("go run useDistributedMutex.go 2 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
		return
	}

	id, _ := strconv.Atoi(os.Args[1])
	addressesAsString := os.Args[2:]
	addresses := make([]PerfectP2PLink.Address, len(addressesAsString))
	for i, address := range addressesAsString {
		addresses[i] = PerfectP2PLink.Address(address)
	}

	var distributedMutex = DistributedMutex.NewDistributedMutexModule(addresses, DistributedMutex.AgentId(id), true)
	fmt.Println(distributedMutex)

	time.Sleep(5 * time.Second)

	for {
		fmt.Println("Application of id: ", id, " requests Mutex")
		distributedMutex.ApplicationRequests <- DistributedMutex.ENTER
		fmt.Println("Application of id: ", id, " waits for Mutex")
		<-distributedMutex.AllowApplicationToAccess //
		fmt.Println("Application of id: ", id, " has the mutex")
		distributedMutex.ApplicationRequests <- DistributedMutex.EXIT //
		fmt.Println("Application of id: ", id, " leaves the mutex")
	}
}
