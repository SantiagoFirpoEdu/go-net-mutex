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
	ApplicationRequests      chan requestFromApplication // canal para receber pedidos da aplicacao (REQ e EXIT)
	AllowApplicationToAccess chan allowAccessMessage     // canal para informar aplicacao que pode acessar
	addresses                []PerfectP2PLink.Address    // endereco de todos, na mesma ordem
	addressToId              map[PerfectP2PLink.Address]AgentId
	id                       AgentId        // identificador do processo - é o indice no array de enderecos acima
	state                    EState         // estado deste processo na exclusao mutua distribuida
	isAgentWaiting           []bool         // processos aguardando tem flag true
	localLogicalClock        localTimestamp // relogio logico local
	responseOkAmount         uint
	isInDebugMode            bool

	PerfectP2PLink *PerfectP2PLink.PerfectP2PLink // acesso aa comunicacao enviar por PP2PLinq.ApplicationRequests  e receber por PP2PLinq.AllowApplicationToAccess
}

// ------------------------------------------------------------------------------------
// ------- inicializacao
// ------------------------------------------------------------------------------------

func NewDistributedMutexModule(addresses []PerfectP2PLink.Address, id AgentId, isDebug bool) *Module {

	distributedMutexModule := &Module{
		ApplicationRequests:      make(chan requestFromApplication, 1),
		AllowApplicationToAccess: make(chan allowAccessMessage, 1),
		addressToId:              make(map[PerfectP2PLink.Address]AgentId),
		addresses:                addresses,
		id:                       id,
		state:                    notInMutualExclusion,
		isAgentWaiting:           make([]bool, len(addresses)),
		localLogicalClock:        0,
		isInDebugMode:            isDebug,

		PerfectP2PLink: PerfectP2PLink.NewLink(addresses[id], isDebug)}

	for i := AgentId(0); i < AgentId(len(distributedMutexModule.isAgentWaiting)); i++ {
		distributedMutexModule.isAgentWaiting[i] = false
		distributedMutexModule.addressToId[addresses[i]] = i
	}

	distributedMutexModule.Start()
	distributedMutexModule.debugLog("Initialized Distributed Mutex!")

	return distributedMutexModule
}

func (module *Module) Start() {
	go func() {
		for {
			select {
			case applicationRequest := <-module.ApplicationRequests: // vindo da  aplicação
				if applicationRequest == ENTER {
					module.debugLog("App", module.id, "is requesting mutual exclusion")
					module.requestEntry()

				} else if applicationRequest == EXIT {
					module.debugLog("App", module.id, "releases mutual exclusion")
					module.requestExit()
				}

			case otherMessage := <-module.PerfectP2PLink.InputChannel:
				otherTimestamp := getTimestamp(otherMessage)
				if strings.Contains(otherMessage.Content, responseOk) {
					module.debugLog("         <<<---- Responds! " + otherMessage.Content)
					module.handleRequestOkFromOther() // ENTRADA DO ALGORITMO

				} else if strings.Contains(otherMessage.Content, requestEntry) {
					module.debugLog("          <<<---- Requests??  " + otherMessage.Content)
					module.handleEntryRequestFromAnother(otherMessage.From, otherTimestamp, module.addressToId[otherMessage.From]) // ENTRADA DO ALGORITMO
				}
			}
		}
	}()
}

func getTimestamp(message PerfectP2PLink.InRequest) localTimestamp {
	split := strings.Split(message.Content, " ")
	timestamp := split[1]
	atoi, err := strconv.Atoi(timestamp)
	if err != nil {
		panic("Unable to get localTimestamp from message")
	}

	return localTimestamp(atoi)
}

func (module *Module) requestEntry() {
	select {
	case request := <-module.ApplicationRequests:
		{
			if request == ENTER {
				module.localLogicalClock++

				for _, address := range module.addresses {
					module.sendRequestEntry(address)
				}

				module.state = wantsMutualExclusion
			}
		}
	}
	/*
					upon event [ dmx, Entry  |  r ]  do
		    			lts.ts++
		    			myTs := lts
		    			resps := 0
		    			para todo processo p
							trigger [ pl , Send | [ reqEntry, r, myTs ]
		    			estado := queroSC
	*/
}

func (module *Module) requestExit() {
	for index, address := range module.addresses {
		if module.isAgentWaiting[index] {
			module.sendResponseOk(address)
		}
	}

	module.state = notInMutualExclusion
	module.isAgentWaiting[module.id] = false
	/*
						upon event [ dmx, Exit  |  r  ]  do
		       				para todo [p, r, ts ] em isAgentWaiting
		          				trigger [ pl, Send | p , [ respOk, r ]  ]
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
		module.AllowApplicationToAccess <- struct{}{}
		module.state = inMutualExclusion
	}
	/*
						upon event [ pl, Deliver | p, [ respOk, r ] ]
		      				resps++
		      				se resps = N
		    				então trigger [ dmx, Deliver | free2Access ]
		  					    estado := estouNaSC

	*/
}

func (module *Module) handleEntryRequestFromAnother(from PerfectP2PLink.Address, otherLogicalClock localTimestamp, otherId AgentId) {
	if module.state == notInMutualExclusion || (module.state == wantsMutualExclusion && isBefore(otherId, otherLogicalClock, module.id, module.localLogicalClock)) {
		module.sendResponseOk(from)
	} else if module.state == inMutualExclusion || (module.state == wantsMutualExclusion && isBefore(module.id, module.localLogicalClock, otherId, otherLogicalClock)) {
	}
	/*
						upon event [ pl, Deliver | p, [ reqEntry, r, rts ]  do
		     				se (estado == naoQueroSC)   OR
		        				 (estado == QueroSC AND  myTs >  ts)
							então  trigger [ pl, Send | p , [ respOk, r ]  ]
		 					senão
		        				se (estado == estouNaSC) OR
		           					 (estado == QueroSC AND  myTs < ts)
		        				então  postergados := postergados + [p, r ]
		     					lts.ts := max(lts.ts, rts.ts)
	*/
}

// ------------------------------------------------------------------------------------
// ------- funcoes de ajuda
// ------------------------------------------------------------------------------------

func (module *Module) sendToLink(address PerfectP2PLink.Address, content string, space string) {
	module.debugLog(space + " ---->>>>   to: " + string(address) + "     msg: " + content)
	module.PerfectP2PLink.OutputChannel <- PerfectP2PLink.OutRequest{
		To:      address,
		Message: content}
}

func (module *Module) sendResponseOk(address PerfectP2PLink.Address) {
	module.sendToLink(address, responseOk+" "+strconv.FormatUint(uint64(module.localLogicalClock), 10), " ")
}

func (module *Module) sendRequestEntry(address PerfectP2PLink.Address) {
	module.sendToLink(address, requestEntry+" "+strconv.FormatUint(uint64(module.localLogicalClock), 10), " ")
}

func isBefore(firstId AgentId, firstLogicalClock localTimestamp, otherId AgentId, otherLogicalClock localTimestamp) bool {
	if firstLogicalClock < otherLogicalClock {
		return true
	} else if firstLogicalClock > otherLogicalClock {
		return false
	}

	return firstId < otherId
}

func (module *Module) debugLog(a ...any) {
	if module.isInDebugMode {
		fmt.Println("[Mutex Debug]", a)
	}
}
