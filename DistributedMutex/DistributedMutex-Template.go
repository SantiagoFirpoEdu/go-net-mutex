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
	"strings"
)

const responseOk = "responseOk"
const requestEntry = "requestEntry"

// ------------------------------------------------------------------------------------
// ------- principais tipos
// ------------------------------------------------------------------------------------

type EState int // enumeracao dos estados possiveis de um processo
const (
	noMutualExclusion EState = iota
	wantsMutualExclusion
	inMutualExclusion
)

type EDmxRequestType int // enumeracao dos estados possiveis de um processo

const (
	ENTER EDmxRequestType = iota
	EXIT
)

type allowAccessMessage struct { // mensagem do módulo DIMEX infrmando que pode acessar - pode ser somente um sinal (vazio)
	// mensagem para aplicacao indicando que pode prosseguir
}

type DistributedMutexModule struct {
	RequestInputChannel  chan EDmxRequestType    // canal para receber pedidos da aplicacao (REQ e EXIT)
	AllowAccessChannel   chan allowAccessMessage // canal para informar aplicacao que pode acessar
	addresses            []string                // endereco de todos, na mesma ordem
	id                   int                     // identificador do processo - é o indice no array de enderecos acima
	mutexState           EState                  // estado deste processo na exclusao mutua distribuida
	isWaitingFlags       []bool                  // processos aguardando tem flag true
	localLogicalClock    int                     // relogio logico local
	lastRequestTimestamp int                     // timestamp local da ultima requisicao deste processo
	responseAmount       int
	isInDebugMode        bool

	PerfectP2PLink *PerfectP2PLink.PerfectP2PLink // acesso aa comunicacao enviar por PP2PLinq.RequestInputChannel  e receber por PP2PLinq.AllowAccessChannel
}

// ------------------------------------------------------------------------------------
// ------- inicializacao
// ------------------------------------------------------------------------------------

func NewDistributedMutexModule(addresses []string, id int, isDebug bool) *DistributedMutexModule {

	distributedMutexModule := &DistributedMutexModule{
		RequestInputChannel: make(chan EDmxRequestType, 1),
		AllowAccessChannel:  make(chan allowAccessMessage, 1),

		addresses:            addresses,
		id:                   id,
		mutexState:           noMutualExclusion,
		isWaitingFlags:       make([]bool, len(addresses)),
		localLogicalClock:    0,
		lastRequestTimestamp: 0,
		isInDebugMode:        isDebug,

		PerfectP2PLink: PerfectP2PLink.NewLink(addresses[id], isDebug)}

	for i := 0; i < len(distributedMutexModule.isWaitingFlags); i++ {
		distributedMutexModule.isWaitingFlags[i] = false
	}

	distributedMutexModule.Start()
	distributedMutexModule.debugLog("Initialized Distributed Mutex!")

	return distributedMutexModule
}

func (module *DistributedMutexModule) Start() {
	go func() {
		for {
			select {
			case applicationRequest := <-module.RequestInputChannel: // vindo da  aplicação
				if applicationRequest == ENTER {
					module.debugLog("App", module.id, "is requesting mutual exclusion")
					module.requestEntry()

				} else if applicationRequest == EXIT {
					module.debugLog("App", module.id, "releases mutual exclusion")
					module.requestExit()
				}

			case otherMessage := <-module.PerfectP2PLink.InputChannel:
				if strings.Contains(otherMessage.Message, responseOk) {
					module.debugLog("         <<<---- Responds! " + otherMessage.Message)
					module.handleRequestOkFromOther(otherMessage) // ENTRADA DO ALGORITMO

				} else if strings.Contains(otherMessage.Message, requestEntry) {
					module.debugLog("          <<<---- Requests??  " + otherMessage.Message)
					module.handleEntryRequestFromAnother(otherMessage) // ENTRADA DO ALGORITMO
				}
			}
		}
	}()
}

func (module *DistributedMutexModule) requestEntry() {
	select {
	case request := <-module.RequestInputChannel:
		{
			if request == ENTER {
				module.lastRequestTimestamp++
				module.lastRequestTimestamp = module.lastRequestTimestamp

				for _, address := range module.addresses {
					module.sendToLink(address, requestEntry, " ")
				}

				module.mutexState = wantsMutualExclusion
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

func (module *DistributedMutexModule) requestExit() {
	select {
	case request := <-module.RequestInputChannel:
		{
			if request == EXIT {
				module.mutexState = noMutualExclusion
			}
		}
	}
	/*
						upon event [ dmx, Exit  |  r  ]  do
		       				para todo [p, r, ts ] em isWaitingFlags
		          				trigger [ pl, Send | p , [ respOk, r ]  ]
		    				estado := naoQueroSC
							isWaitingFlags := {}
	*/
}

// ------------------------------------------------------------------------------------
// ------- tratamento de mensagens de outros processos
// ------- UPON respOK
// ------- UPON reqEntry
// ------------------------------------------------------------------------------------

func (module *DistributedMutexModule) handleRequestOkFromOther(msgOutro PerfectP2PLink.InRequest) {
	module.responseAmount++
	if module.responseAmount == len(module.addresses) {
		module.sendToLink()
	}
	/*
						upon event [ pl, Deliver | p, [ respOk, r ] ]
		      				resps++
		      				se resps = N
		    				então trigger [ dmx, Deliver | free2Access ]
		  					    estado := estouNaSC

	*/
}

func (module *DistributedMutexModule) handleEntryRequestFromAnother(otherMessage PerfectP2PLink.InRequest) {
	// outro processo quer entrar na SC
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

func (module *DistributedMutexModule) sendToLink(address string, content string, space string) {
	module.debugLog(space + " ---->>>>   to: " + address + "     msg: " + content)
	module.PerfectP2PLink.OutputChannel <- PerfectP2PLink.OutRequest{
		To:      address,
		Message: content}
}

func before(oneId, oneTs, othId, othTs int) bool {
	if oneTs < othTs {
		return true
	} else if oneTs > othTs {
		return false
	} else {
		return oneId < othId
	}
}

func (module *DistributedMutexModule) debugLog(a ...any) {
	if module.isInDebugMode {
		fmt.Println("[Mutex Debug]", a)
	}
}
