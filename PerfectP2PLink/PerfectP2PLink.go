/*
  Construido como parte da disciplina: Sistemas Distribuidos - PUCRS - Escola Politecnica
  Professor: Fernando Dotti  (https://fldotti.github.io/)
  Modulo representando Perfect Point to Point Links tal como definido em:
    Introduction to Reliable and Secure Distributed Programming
    Christian Cachin, Rachid Gerraoui, Luis Rodrigues
  * Semestre 2018/2 - Primeira versao.  Estudantes:  Andre Antonitsch e Rafael Copstein
  * Semestre 2019/1 - Reaproveita conexões TCP já abertas - Estudantes: Vinicius Sesti e Gabriel Waengertner
  * Semestre 2020/1 - Separa mensagens de qualquer tamanho atee 4 digitos.
  Sender envia tamanho no formato 4 digitos (preenche com 0s a esquerda)
  Receiver recebe 4 digitos, calcula tamanho do buffer a receber,
  e recebe com io.ReadFull o tamanho informado - Dotti
  * Semestre 2022/1 - melhorias eliminando retorno de erro aos canais superiores.
  se conexao fecha nao retorna nada.   melhorias em comentarios.   adicionado modo debug. - Dotti
*/

package PerfectP2PLink

import (
	"fmt"
	"io"
	"net"
	"strconv"
)

type OutRequest struct {
	To      string
	Message string
}

type InRequest struct {
	From    string
	Message string
}

type PerfectP2PLink struct {
	InputChannel  chan InRequest
	OutputChannel chan OutRequest
	ShouldRun     bool
	isInDebugMode bool
	Cache         map[string]net.Conn // cache de conexoes - reaproveita conexao com destino ao inves de abrir outra
}

func NewLink(_address string, _dbg bool) *PerfectP2PLink {
	p2p := &PerfectP2PLink{
		OutputChannel: make(chan OutRequest, 1),
		InputChannel:  make(chan InRequest, 1),
		ShouldRun:     true,
		isInDebugMode: _dbg,
		Cache:         make(map[string]net.Conn)}
	p2p.debugLog(" Init PerfectP2PLink!")
	p2p.Start(_address)
	return p2p
}

func (module *PerfectP2PLink) debugLog(s string) {
	if module.isInDebugMode {
		fmt.Println(". . . . . . . . . . . . . . . . . [ PerfectP2PLink msg : " + s + " ]")
	}
}

func (module *PerfectP2PLink) Start(address string) {

	// PROCESSO PARA RECEBIMENTO DE MENSAGENS
	go func() {
		listen, _ := net.Listen("tcp4", address)
		for {
			// aceita repetidamente tentativas novas de conexao
			conn, err := listen.Accept()
			module.debugLog("ok   : conexao aceita com outro processo.")
			// para cada conexao lanca rotina de tratamento
			go func() {
				// repetidamente recebe mensagens na conexao TCP (sem fechar)
				// e passa para modulo de cima
				for { //                              // enquanto conexao aberta
					if err != nil {
						fmt.Println(".", err)
						break
					}
					sizeBuffer := make([]byte, 4) //       // le tamanho da mensagem
					_, err := io.ReadFull(conn, sizeBuffer)
					if err != nil {
						module.debugLog("erro : " + err.Error() + " conexao fechada pelo outro processo.")
						break
					}
					size, err := strconv.Atoi(string(sizeBuffer))
					bufMsg := make([]byte, size)       // declara buffer do tamanho exato
					_, err = io.ReadFull(conn, bufMsg) // le do tamanho do buffer ou da erro
					if err != nil {
						fmt.Println("@", err)
						break
					}
					message := InRequest{
						From:    conn.RemoteAddr().String(),
						Message: string(bufMsg)}
					// ATE AQUI:  procedimentos para receber message
					module.InputChannel <- message //               // repassa mensagem para modulo superior
				}
			}()
		}
	}()

	// PROCESSO PARA ENVIO DE MENSAGENS
	go func() {
		for {
			message := <-module.OutputChannel
			module.Send(message)
		}
	}()
}

func (module *PerfectP2PLink) Send(message OutRequest) {
	var connection net.Conn
	var ok bool
	var err error

	// ja existe uma conexao aberta para aquele destinatario?
	if connection, ok = module.Cache[message.To]; ok {
	} else { // se nao existe, abre e guarda na cache
		connection, err = net.Dial("tcp", message.To)
		module.debugLog("ok   : conexao iniciada com outro processo")
		if err != nil {
			fmt.Println(err)
			return
		}
		module.Cache[message.To] = connection
	}
	// calcula tamanho da mensagem e monta string de 4 caracteres numericos com o tamanho.
	// completa com 0s aa esquerda para fechar tamanho se necessario.
	messageString := strconv.Itoa(len(message.Message))
	for len(messageString) < 4 {
		messageString = "0" + messageString
	}
	if !(len(messageString) == 4) {
		module.debugLog("ERROR AT PPLINK MESSAGE SIZE CALCULATION - INVALID MESSAGES MAY BE IN TRANSIT")
	}
	_, err = fmt.Fprintf(connection, messageString)   // escreve 4 caracteres com tamanho
	_, err = fmt.Fprintf(connection, message.Message) // escreve a mensagem com o tamanho calculado
	if err != nil {
		module.debugLog("erro : " + err.Error() + ". Conexao fechada. 1 tentativa de reabrir:")
		connection, err = net.Dial("tcp", message.To)
		if err != nil {
			module.debugLog("       " + err.Error())
			return
		} else {
			module.debugLog("ok   : conexao iniciada com outro processo.")
		}
		module.Cache[message.To] = connection
		_, err = fmt.Fprintf(connection, messageString)   // escreve 4 caracteres com tamanho
		_, err = fmt.Fprintf(connection, message.Message) // escreve a mensagem com o tamanho calculado
	}
	return
}
