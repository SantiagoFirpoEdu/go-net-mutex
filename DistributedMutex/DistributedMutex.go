/*  Construido como parte da disciplina: FPPD - PUCRS - Escola Politecnica
    Professor: Fernando Dotti  (https://fldotti.github.io/)
    Modulo representando Algoritmo de Exclusão Mútua Distribuída:
    Semestre 2023/1
	Aspectos a observar:
	   mapeamento de módulo para estrutura
	   inicializacao
	   semantica de concorrência: cada evento é atômico
	   							  módulo trata 1 por vez
	Q U E S T A O
	   Além de obviamente entender a estrutura ...
	   Implementar o núcleo do algoritmo ja descrito, ou seja, o corpo das
	   funcoes reativas a cada entrada possível:
	   			requestEntry()  // recebe do nivel de cima (app)
				requestExit()   // recebe do nivel de cima (app)
				handleRequestOkFromOther(msgOutro)   // recebe do nivel de baixo
				handleEntryRequestFromAnother(msgOutro) // recebe do nivel de baixo
*/

package DistributedMutex

import (
	"SD/PerfectP2PLink"
	"fmt"
	"strconv"
	"strings"
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

const responseOk = "responseOk"
const requestEntry = "requestEntry"

// ------------------------------------------------------------------------------------
// ------- principais tipos
// ------------------------------------------------------------------------------------

type EState int // enumeracao dos estados possiveis de um processo
type localTimestamp uint64
type AgentId uint64

const (
	notInMutualExclusion EState = iota
	wantsMutualExclusion
	inMutualExclusion
)

type requestFromApplication int // enumeracao dos estados possiveis de um processo

const (
	ENTER requestFromApplication = iota
	EXIT
)

type allowAccessMessage struct { // mensagem do módulo DIMEX infrmando que pode acessar - pode ser somente um sinal (vazio)
	// mensagem para aplicacao indicando que pode prosseguir
}

type Module struct {
	applicationRequests       chan requestFromApplication
	applicationIsFreeToAccess chan allowAccessMessage
	addresses                 []PerfectP2PLink.Address
	addressToIndex            map[PerfectP2PLink.Address]AgentId
	index                     AgentId
	state                     EState
	isAgentWaiting            []bool
	timestamp                 localTimestamp
	responseOkAmount          uint
	isInDebugMode             bool

	PerfectP2PLink *PerfectP2PLink.PerfectP2PLink
}

func NewDistributedMutexModule(addresses []PerfectP2PLink.Address, id AgentId, isDebug bool) *Module {

	distributedMutexModule := &Module{
		applicationRequests:       make(chan requestFromApplication, 1),
		applicationIsFreeToAccess: make(chan allowAccessMessage, 1),
		addressToIndex:            make(map[PerfectP2PLink.Address]AgentId),
		addresses:                 addresses,
		index:                     id,
		state:                     notInMutualExclusion,
		isAgentWaiting:            make([]bool, len(addresses)),
		timestamp:                 0,
		isInDebugMode:             isDebug,

		PerfectP2PLink: PerfectP2PLink.NewLink(addresses[id], isDebug)}

	for i := AgentId(0); i < AgentId(len(distributedMutexModule.isAgentWaiting)); i++ {
		distributedMutexModule.isAgentWaiting[i] = false
		distributedMutexModule.addressToIndex[addresses[i]] = i
	}

	distributedMutexModule.Start()
	distributedMutexModule.debugLog("Initialized Distributed Mutex!")

	return distributedMutexModule
}

func (module *Module) Start() {
	go func() {
		for {
			select {
			case applicationRequest := <-module.applicationRequests:
				if applicationRequest == ENTER {
					module.debugLog("App", module.index, "is requesting mutual exclusion")
					module.requestEntry()

				} else if applicationRequest == EXIT {
					module.debugLog("App", module.index, "releases mutual exclusion")
					module.requestExit()
				}

			case messageFromAnotherModule := <-module.PerfectP2PLink.ReceiveChannel():
				otherTimestamp := getTimestamp(messageFromAnotherModule)
				if strings.Contains(messageFromAnotherModule.Content, responseOk) {
					module.debugLog("Received message from the module", module.addressToIndex[messageFromAnotherModule.From], "with a ResponseOk:", messageFromAnotherModule.Content)
					module.handleRequestOkFromOther()

				} else if strings.Contains(messageFromAnotherModule.Content, requestEntry) {
					module.debugLog("Received message from the module", module.addressToIndex[messageFromAnotherModule.From], "with a RequestEntry:", messageFromAnotherModule.Content)
					module.handleEntryRequestFromAnother(messageFromAnotherModule.From, otherTimestamp, module.addressToIndex[messageFromAnotherModule.From]) // ENTRADA DO ALGORITMO
				}
			}
		}
	}()
}

func getTimestamp(message PerfectP2PLink.ReceivedRequest) localTimestamp {
	split := strings.Split(message.Content, " ")
	timestamp := split[1]
	atoi, err := strconv.Atoi(timestamp)
	if err != nil {
		panic("Unable to get localTimestamp from message")
	}

	return localTimestamp(atoi)
}

func (module *Module) requestEntry() {
	module.timestamp++

	for index, address := range module.addresses {
		if module.index != AgentId(index) {
			module.sendRequestEntry(address)
		}
	}

	module.state = wantsMutualExclusion
	/*
		lts.ts++
		myTs := lts
		resps := 0
		para todo processo p
			trigger [ pl , SentRequest | [ reqEntry, r, myTs ]
		estado := queroSC
	*/
}

func (module *Module) requestExit() {
	for index, address := range module.addresses {
		if module.isAgentWaiting[index] {
			module.sendOkResponse(address)
		}
	}

	module.state = notInMutualExclusion
	module.isAgentWaiting[module.index] = false
	/*
						upon event [ dmx, Exit  |  r  ]  do
		       				para todo [p, r, ts ] em isAgentWaiting
		          				trigger [ pl, SentRequest | p , [ respOk, r ]  ]
		    				estado := naoQueroSC
							isAgentWaiting := {}
	*/
}

// ------------------------------------------------------------------------------------
// ------- tratamento de mensagens de outros processos
// ------- UPON respOK
// ------- UPON reqEntry
// ------------------------------------------------------------------------------------

func (module *Module) handleRequestOkFromOther() {
	module.responseOkAmount++
	if module.responseOkAmount == uint(len(module.addresses)) {
		module.sendFreeToAccess()
	}
	/*
		resps++
		se resps = N
		então trigger [ dmx, Deliver | free2Access ]
			estado := estouNaSC

	*/
}

func (module *Module) sendFreeToAccess() {
	module.applicationIsFreeToAccess <- struct{}{}
	module.state = inMutualExclusion
}

func (module *Module) handleEntryRequestFromAnother(from PerfectP2PLink.Address, otherTimestamp localTimestamp, otherId AgentId) {
	isOtherBefore := isBefore(otherId, otherTimestamp, module.index, module.timestamp)
	isOtherAfter := isBefore(module.index, module.timestamp, otherId, otherTimestamp)

	shouldSendOkResponse := module.state == notInMutualExclusion || (module.state == wantsMutualExclusion && isOtherBefore)
	shouldDelayResponse := module.state == inMutualExclusion || (module.state == wantsMutualExclusion && isOtherAfter)

	if shouldSendOkResponse {
		module.sendOkResponse(from)
	} else if shouldDelayResponse {
		module.isAgentWaiting[module.index] = true
		module.timestamp = maxTimestamp(module.timestamp, otherTimestamp)
	}
	/*
		se (estado == naoQueroSC)   OR
			 (estado == QueroSC AND  myTs >  ts)
		então  trigger [ pl, SentRequest | p , [ respOk, r ]  ]
		senão
			se (estado == estouNaSC) OR
				 (estado == QueroSC AND  myTs < ts)
			então  postergados := postergados + [p, r ]
			lts.ts := maxTimestamp(lts.ts, rts.ts)
	*/
}

func maxTimestamp(first localTimestamp, second localTimestamp) localTimestamp {
	if first > second {
		return first
	}
	return second
}

func (module *Module) sendToLink(address PerfectP2PLink.Address, content string) {
	module.debugLog(module.index, "--> to module:", module.addressToIndex[address], ". message:", content)
	module.PerfectP2PLink.SendChannel() <- PerfectP2PLink.SentRequest{
		To:      address,
		Message: content}
}

func (module *Module) sendOkResponse(address PerfectP2PLink.Address) {
	module.sendToLink(address, responseOk+" "+strconv.FormatUint(uint64(module.timestamp), 10))
}

func (module *Module) sendRequestEntry(address PerfectP2PLink.Address) {
	module.sendToLink(address, requestEntry+" "+strconv.FormatUint(uint64(module.timestamp), 10))
}

func isBefore(firstId AgentId, firstTimestamp localTimestamp, otherId AgentId, otherTimestamp localTimestamp) bool {
	if firstTimestamp == otherTimestamp {
		return firstId < otherId
	}

	return firstTimestamp < otherTimestamp
}

func (module *Module) ApplicationRequests() chan<- requestFromApplication {
	return module.applicationRequests
}

func (module *Module) ApplicationIsFreeToAccess() <-chan allowAccessMessage {
	return module.applicationIsFreeToAccess
}

func (module *Module) debugLog(a ...any) {
	if module.isInDebugMode {
		fmt.Println(Blue, "[Mutex Debug - Module: ", module.index, "]", a, Reset)
	}
}
