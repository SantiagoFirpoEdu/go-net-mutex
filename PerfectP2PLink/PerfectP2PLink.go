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

const Reset = "\033[0m"
const Red = "\033[31m"
const Green = "\033[32m"
const Yellow = "\033[33m"
const Blue = "\033[34m"
const Magenta = "\033[35m"
const Cyan = "\033[36m"
const Gray = "\033[37m"
const White = "\033[97m"

type Address string

type SentRequest struct {
	To      Address
	Message string
}

type ReceivedRequest struct {
	From    Address
	Content string
}

type PerfectP2PLink struct {
	receiveChannel    chan ReceivedRequest
	sendChannel       chan SentRequest
	ShouldRun         bool
	isInDebugMode     bool
	CachedConnections map[Address]net.Conn
}

func NewLink(_address Address, _dbg bool) *PerfectP2PLink {
	p2p := &PerfectP2PLink{
		sendChannel:       make(chan SentRequest, 1),
		receiveChannel:    make(chan ReceivedRequest, 1),
		ShouldRun:         true,
		isInDebugMode:     _dbg,
		CachedConnections: make(map[Address]net.Conn)}
	p2p.debugLog(" Init PerfectP2PLink!")
	p2p.Start(_address)
	return p2p
}

func (module *PerfectP2PLink) debugLog(s string) {
	if module.isInDebugMode {
		fmt.Println(Green, "[ PerfectP2PLink msg : "+s+" ]", Reset)
	}
}

func (module *PerfectP2PLink) Start(address Address) {
	go func() {
		listen, _ := net.Listen("tcp4", string(address))
		for {
			conn, err := listen.Accept()
			module.debugLog("ok   : connection accepted with another process.")
			go func() {
				for {
					if err != nil {
						fmt.Println(".", err)
						break
					}
					sizeBuffer := make([]byte, 4)
					_, err := io.ReadFull(conn, sizeBuffer)
					if err != nil {
						module.debugLog("erro : " + err.Error() + " connection closed by another process.")
						break
					}
					size, err := strconv.Atoi(string(sizeBuffer))
					bufMsg := make([]byte, size)
					_, err = io.ReadFull(conn, bufMsg)
					if err != nil {
						fmt.Println("@", err)
						break
					}
					message := ReceivedRequest{
						From:    Address(conn.RemoteAddr().String()),
						Content: string(bufMsg)}
					module.receiveChannel <- message
				}
			}()
		}
	}()

	go func() {
		for {
			message := <-module.sendChannel
			module.send(message)
		}
	}()
}

func (module *PerfectP2PLink) ReceiveChannel() <-chan ReceivedRequest {
	return module.receiveChannel
}

func (module *PerfectP2PLink) send(message SentRequest) {
	var connection net.Conn
	var ok bool
	var err error

	if connection, ok = module.CachedConnections[message.To]; ok {
	} else {
		connection, err = net.Dial("tcp", string(message.To))
		module.debugLog("ok   : connection established with another process")
		if err != nil {
			fmt.Println(err)
			return
		}
		module.CachedConnections[message.To] = connection
	}
	messageString := strconv.Itoa(len(message.Message))
	for len(messageString) < 4 {
		messageString = "0" + messageString
	}
	if !(len(messageString) == 4) {
		module.debugLog("ERROR AT PPLINK MESSAGE SIZE CALCULATION - INVALID MESSAGES MAY BE IN TRANSIT")
	}
	_, err = fmt.Fprintf(connection, messageString)
	_, err = fmt.Fprintf(connection, message.Message)
	if err != nil {
		module.debugLog("erro : " + err.Error() + ". Connection closed. 1 attempt to reopen:")
		connection, err = net.Dial("tcp", string(message.To))
		if err != nil {
			module.debugLog("       " + err.Error())
			return
		} else {
			module.debugLog("ok   : connection established with another process.")
		}
		module.CachedConnections[message.To] = connection
		_, err = fmt.Fprintf(connection, messageString)
		_, err = fmt.Fprintf(connection, message.Message)
	}
	return
}

func (module *PerfectP2PLink) SendChannel() chan<- SentRequest {
	return module.sendChannel
}
