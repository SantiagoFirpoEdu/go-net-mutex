// Construido como parte da disciplina: Sistemas Distribuidos - PUCRS - Escola Politecnica
//  Professor: Fernando Dotti  (https://fldotti.github.io/)
// Uso p exemplo:
//   go run usaDIMEX.go 0 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
//   go run usaDIMEX.go 1 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
//   go run usaDIMEX.go 2 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
// ----------
// LANCAR N PROCESSOS EM SHELL's DIFERENTES, UMA PARA CADA PROCESSO.
// para cada processo fornecer: seu id único (0, 1, 2 ...) e a mesma lista de processos.
// o endereco de cada processo é o dado na lista, na posicao do seu id.
// no exemplo acima o processo com id=1  usa a porta 6001 para receber e as portas
// 5000 e 7002 para mandar mensagens respectivamente para processos com id=0 e 2
// -----------
// Esta versão supõe que todos processos tem acesso a um mesmo arquivo chamado "mxOUT.txt"
// Todos processos escrevem neste arquivo, usando o protocolo dimex para exclusao mutua.
// Os processos escrevem "|." cada vez que acessam o arquivo.   Assim, o arquivo com conteúdo
// correto deverá ser uma sequencia de
// |.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.
// |.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.
// |.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.
// etc etc ...     ....  até o usuário interromper os processos (ctl c).
// Qualquer padrao diferente disso, revela um erro.
//      |.|.|.|.|.||..|.|.|.  etc etc  por exemplo.
// Se voce retirar o protocolo dimex vai ver que o arquivo poderá entrelacar
// "|."  dos processos de diversas diferentes formas.
// Ou seja, o padrão correto acima é garantido pelo dimex.
// Ainda assim, isto é apenas um teste.  E testes são frágeis em sistemas distribuídos.

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
		fmt.Println("go run useDistributedMutex-File.go 0 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
		fmt.Println("go run useDistributedMutex-File.go 1 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
		fmt.Println("go run useDistributedMutex-File.go 2 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
		return
	}

	id, _ := strconv.Atoi(os.Args[1])
	addressArgs := os.Args[2:]
	addresses := make([]PerfectP2PLink.Address, len(addressArgs))
	for i, address := range addressArgs {
		addresses[i] = PerfectP2PLink.Address(address)
	}

	distributedMutex := DistributedMutex.NewDistributedMutexModule(addresses, DistributedMutex.AgentId(id), true)

	file, err := os.OpenFile("./mxOUT.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	time.Sleep(3 * time.Second)

	for {
		fmt.Println("[App id: ", id, " asking for mutual exclusion]")
		distributedMutex.ApplicationRequests() <- DistributedMutex.ENTER
		<-distributedMutex.ApplicationIsFreeToAccess()

		_, err = file.WriteString("|")
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}

		fmt.Println("[App id: ", id, " in mutual exclusion]")

		_, err = file.WriteString(".")
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}

		distributedMutex.ApplicationRequests() <- DistributedMutex.EXIT
		fmt.Println("[App id: ", id, "is leaving the mutual exclusion]")
	}
}
