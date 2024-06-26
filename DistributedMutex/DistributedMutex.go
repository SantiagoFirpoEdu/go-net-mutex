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
	   			handleEntryRequestFromApplication()  // recebe do nivel de cima (app)
				handleExitRequestedFromApplication()   // recebe do nivel de cima (app)
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
	doesntWantMutualExclusion EState = iota
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
	localLogicalClock         localTimestamp
	timestamp                 localTimestamp
	okResponseAmount          uint
	isInDebugMode             bool

	PerfectP2PLink *PerfectP2PLink.PerfectP2PLink
}

func NewDistributedMutexModule(addresses []PerfectP2PLink.Address, id AgentId, isDebug bool) *Module {

	distributedMutexModule := &Module{
		applicationRequests:       make(chan requestFromApplication),
		applicationIsFreeToAccess: make(chan allowAccessMessage),
		addressToIndex:            make(map[PerfectP2PLink.Address]AgentId),
		addresses:                 addresses,
		index:                     id,
		state:                     doesntWantMutualExclusion,
		isAgentWaiting:            make([]bool, len(addresses)),
		timestamp:                 0,
		isInDebugMode:             isDebug,

		PerfectP2PLink: PerfectP2PLink.NewLink(addresses[id], true)}

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
					module.handleEntryRequestFromApplication()

				} else if applicationRequest == EXIT {
					module.okResponseAmount = 0
					module.debugLog("App", module.index, "releases mutual exclusion")
					module.handleExitRequestedFromApplication()
				}

			case messageFromAnotherModule := <-module.PerfectP2PLink.ReceiveChannel():
				otherTimestamp := getTimestamp(messageFromAnotherModule)
				split := strings.Split(messageFromAnotherModule.Content, " ")
				otherIdToken := split[2]
				otherIdInt, _ := strconv.Atoi(otherIdToken)
				otherId := AgentId(otherIdInt)
				requestType := split[0]

				if requestType == responseOk {
					module.debugLog("Received message from the module", otherId, "with a ResponseOk. Timestamp is", otherTimestamp)
					module.handleRequestOkFromOther()

				} else if requestType == requestEntry {
					module.debugLog("Received message from the module", otherId, "with a RequestEntry. Timestamp is", otherTimestamp)
					module.handleEntryRequestFromAnother(otherTimestamp, otherId)
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

func (module *Module) handleEntryRequestFromApplication() {
	module.localLogicalClock++
	module.timestamp = module.localLogicalClock
	module.okResponseAmount = 0

	for index := range module.addresses {
		if module.index != AgentId(index) {
			module.sendRequestEntry(AgentId(index))
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

func (module *Module) handleExitRequestedFromApplication() {
	for index := range module.addresses {
		if module.isAgentWaiting[index] {
			module.sendOkResponse(AgentId(index))
			module.isAgentWaiting[index] = false
		}
	}

	module.state = doesntWantMutualExclusion
	/*
						upon event [ dmx, Exit  |  r  ]  do
		       				para todo [p, r, ts ] em isAgentWaiting
		          				trigger [ pl, SentRequest | p , [ respOk, r ]  ]
		    				estado := naoQueroSC
							isAgentWaiting := {}
	*/
}

func (module *Module) handleRequestOkFromOther() {
	module.okResponseAmount++
	if module.okResponseAmount == uint(len(module.addresses)-1) {
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

func (module *Module) handleEntryRequestFromAnother(otherTimestamp localTimestamp, otherId AgentId) {
	isThisAfter := isAfter(module.index, module.timestamp, otherId, otherTimestamp)

	shouldSendOkResponse := module.state == doesntWantMutualExclusion || (module.state == wantsMutualExclusion && isThisAfter)

	if shouldSendOkResponse {
		module.sendOkResponse(otherId)
	} else {
		module.isAgentWaiting[otherId] = true
	}

	module.localLogicalClock = maxTimestamp(module.localLogicalClock, otherTimestamp)
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

func (module *Module) sendToLink(otherId AgentId, content string) {
	module.debugLog(module.index, "--> to module:", otherId, ". message:", content)
	module.PerfectP2PLink.SendChannel() <- PerfectP2PLink.SentRequest{
		To:      module.addresses[otherId],
		Message: content}
}

func (module *Module) sendOkResponse(otherId AgentId) {
	module.sendToLink(otherId, responseOk+" "+strconv.FormatUint(uint64(module.timestamp), 10)+" "+strconv.FormatUint(uint64(module.index), 10))
}

func (module *Module) sendRequestEntry(otherId AgentId) {
	module.sendToLink(otherId, requestEntry+" "+strconv.FormatUint(uint64(module.timestamp), 10)+" "+strconv.FormatUint(uint64(module.index), 10))
}

func isAfter(firstId AgentId, firstTimestamp localTimestamp, otherId AgentId, otherTimestamp localTimestamp) bool {
	if firstTimestamp == otherTimestamp {
		return firstId > otherId
	}

	return firstTimestamp > otherTimestamp
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
