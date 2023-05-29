package main

import (
	"fmt"
	"sync"
	"time"
)

type Message struct {
	counter          int      // Campo usado para controle de voltas no anel, é usado como um contador para mensagens normais, e apenas 0 e 1 para mensagens de eleição
	novoLider        int      // Novo lider, campo usado para reconhecimento do novo lider
	numeroMsg        int      // Numero da mensagem , por motivos de log
	iniciadorEleicao int      // Node que iniciou a eleição
	originalMessage  *Message // Mensagem retida, quando uma eleição interrompe a mesma
	isDelayed        bool     // Boolean que identifica uma mensagem retida pela eleição
	tipo             int      // tipo da mensagem para fazer o controle do que fazer (mensagem padrão, eleição, confirmacao da eleicao)
	corpo            [5]int   // conteudo da mensagem para colocar os ids (usar um tamanho ocmpativel com o numero de processos no anel)
}

type Node struct {
	id        int
	input     chan Message
	output    chan Message
	leaderId  int
	isEnabled bool
}

var (
	// vetor de canais para formar o anel de eleicao - chan[0], chan[1] and chan[2] ...
	chans = []chan Message{
		make(chan Message),
		make(chan Message),
		make(chan Message),
		make(chan Message),
	}
	retorno     = make(chan int)
	nodesAtivos = 4
	waitEleicao = 0
	liderIndex  = 0
	wg          sync.WaitGroup // wg is used to wait for the program to finish
)

func (n *Node) run() {
	for {
		select {
		case message := <-n.input:
			{
				if !n.isEnabled {
					if message.tipo == 1 {
						fmt.Println("Node 1 passando mensagem de eleição para frente")
					} else {
						fmt.Printf("Node %d DESABILITADO recebeu a mensagem e passou pra frente\n", n.id)
					}
					n.output <- message
				} else {
					switch message.tipo {
					case 0: // Mensagem padrão
						{
							if n.id == n.leaderId {
								// sou o lider e conter = 0, a mensagem acabou de entrar no anel
								if message.counter == 0 {
									message.counter = 1
									fmt.Printf("Node %d (Lider) recebeu a mensagem e passou pra frente\n", n.id)
									n.output <- message
								}
							} else {
								// Não sou o lider e a mensagem chegou com counter = 0, portanto lider inativo
								if message.counter == 0 {
									retorno <- -5

									fmt.Printf("Node %d INICIANDO ELEICAO \n", n.id)
									newCorpo := [5]int{0, 0, 0, 0, 0}
									eleicao := Message{
										counter:          1,
										tipo:             1,
										iniciadorEleicao: n.id,
										originalMessage:  &message,
										corpo:            newCorpo,
									}
									eleicao.corpo[n.id] = n.id
									n.output <- eleicao
								} else {
									// Tudo certo, mensagem inicializada pelo lider. Apenas passo pra frente
									fmt.Printf("Node %d consumiu a mensagem e passou pra frente\n", n.id)
									message.counter = message.counter + 1
									// Mensagem completou uma volta, remover do anel em vez de voltar para o lider
									if message.counter == nodesAtivos {
										fmt.Printf("Mensagem %d FINALIZADA\n", message.numeroMsg)
										// Ao fim de uma mensagem que ficou retida, informar controle sobre o novo lider
										if message.isDelayed {
											retorno <- n.leaderId
										} else {
											retorno <- 0
										}
									} else {
										n.output <- message
									}
								}
							}
						}
					case 1: // Mensagem de eleição
						{
							if n.id == n.leaderId {
								fmt.Printf("Algo deu muito errado, ELEICAO COM LIDER ATIVO!!")
								n.output <- message
							} else {
								fmt.Printf("Mensagem ELEICAO %d, %d \n", message.iniciadorEleicao, n.id)
								if message.iniciadorEleicao == n.id {
									fmt.Printf("Mensagem de eleição circulou o anel\n")

									var novoLider = 0

									// using for loop
									for i := 0; i < len(message.corpo); i++ {
										if message.corpo[i] > novoLider {
											novoLider = message.corpo[i]
										}
									}

									fmt.Printf("Novo Lider selecionado, Node %d !!\n", novoLider)

									n.leaderId = novoLider
									reportLider := Message{
										counter:         0,
										tipo:            2,
										novoLider:       novoLider,
										originalMessage: message.originalMessage,
									}

									n.output <- reportLider
								} else {
									fmt.Printf("Node %d votou pra eleição!!\n", n.id)
									message.corpo[n.id] = n.id
									n.output <- message
								}
							}
						}
					case 2: // Mensagem de reconhecimento do Líder
						{
							n.leaderId = message.novoLider
							if n.id == n.leaderId {
								if message.counter == 1 {
									// Mensagem de reconhecimento já girou o anel, remover
									fmt.Printf("Processo de reconhecimento finalizado, todos os nodes já reconhecem %d como lider\n", n.id)
									messageContinue := *message.originalMessage
									messageContinue.counter = 1
									messageContinue.isDelayed = true
									n.output <- messageContinue
								} else {
									message.counter = 1
									n.output <- message
								}
							} else {
								fmt.Printf("Node %d Reconhece o Novo Lider -> Node %d !!\n", n.id, message.novoLider)
								n.output <- message
							}
						}
					default:
						{
						}
					}
				}
			}
		default:
			{
			}
		}
	}
}

func controle(n1 *Node, n2 *Node, n3 *Node, n4 *Node) {
	defer wg.Done()
	i := 0
	for {
		select {
		case message := <-retorno:
			if message == 0 {
				fmt.Printf("Mensagem encerrada com sucesso \n")
			} else if message == -5 {
				fmt.Printf("Controle informado que uma eleição está ativa, aguardar o fim da eleição \n")
				waitEleicao = 1
			} else {
				fmt.Printf("Controle informado sobre o fim da eleição, resumir o envio ao canal \n")
				liderIndex = message - 1
				waitEleicao = 0
			}
		default:
			time.Sleep(1 * time.Second)
			if waitEleicao == 0 {

				// Desabilita node 1 apos 10 rodadas
				if i == 10 {
					n1.isEnabled = false
					nodesAtivos = 3
				}
				// Desabilita node 4 apos 20 rodadas
				// Node 4 era o então lider, deve ser desativado e após eleição novo lider deve ser o node 3
				if i == 20 {
					n4.isEnabled = false
					nodesAtivos = 2
				}

				// Encerra o programa após 30 iterações
				if i == 30 {
					return
				}

				chans[liderIndex] <- Message{tipo: 0, numeroMsg: i}
				i = i + 1
			}
		}
	}
}

func main() {
	wg.Add(1)

	// Create nodes
	node1 := &Node{id: 1, input: chans[0], output: chans[1], isEnabled: true, leaderId: 1}
	node2 := &Node{id: 2, input: chans[1], output: chans[2], isEnabled: true, leaderId: 1}
	node3 := &Node{id: 3, input: chans[2], output: chans[3], isEnabled: true, leaderId: 1}
	node4 := &Node{id: 4, input: chans[3], output: chans[0], isEnabled: true, leaderId: 1}

	go node1.run()
	go node2.run()
	go node3.run()
	go node4.run()

	go controle(node1, node2, node3, node4)

	wg.Wait()

	fmt.Println("Fim do programa de controle, execução completa com sucesso")
}
