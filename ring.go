package main

import (
	"fmt"
	"sync"
	"time"
)

type Process struct {
	id    int
	next  *Process
	token chan int
	done  chan bool
}

func NewProcess(id int) *Process {
	return &Process{
		id:    id,
		token: make(chan int),
		done:  make(chan bool),
	}
}

func (p *Process) SetNext(next *Process) {
	p.next = next
}

func (p *Process) Run(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case token, ok := <-p.token:
			if !ok {
				fmt.Printf("Process %d exiting\n", p.id)
				return
			}
			fmt.Printf("Process %d received token %d\n", p.id, token)

			if token >= p.id {
				time.Sleep(500 * time.Millisecond)
				p.next.token <- token
			} else {
				time.Sleep(500 * time.Millisecond)
				p.next.token <- p.id
			}
		case <-p.done:
			fmt.Printf("Process %d exiting\n", p.id)
			return
		}
	}
}

func main() {
	var wg sync.WaitGroup

	processCount := 5
	processes := make([]*Process, processCount)

	for i := 0; i < processCount; i++ {
		processes[i] = NewProcess(i + 1)
	}

	for i := 0; i < processCount; i++ {
		if i == processCount-1 {
			processes[i].SetNext(processes[0])
		} else {
			processes[i].SetNext(processes[i+1])
		}
	}

	for i := 0; i < processCount; i++ {
		wg.Add(1)
		go processes[i].Run(&wg)
	}

	processes[0].token <- processes[0].id

	time.Sleep(10 * time.Second)

	for _, process := range processes {
		process.done <- true
	}

	wg.Wait()
}
