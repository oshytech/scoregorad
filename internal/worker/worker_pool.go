package worker

import (
	"context"
	"errors"
	"log"
	"sync"
)

// ErrPoolFull se devuelve cuando el canal de trabajos está lleno.
// Esto es backpressure deliberada: el handler HTTP nunca bloquea esperando
// que el pool tenga espacio. Si el pool está saturado, el efecto secundario
// se descarta y el cliente recibe igualmente su respuesta 201.
var ErrPoolFull = errors.New("worker pool is full")

// Processor es la función que ejecuta cada worker al recibir un evento.
type Processor func(ScoreEvent) error

// Pool gestiona un conjunto fijo de goroutines que consumen ScoreEvents
// de un canal compartido.
//
// Diseño clave:
//   - jobs: canal con buffer. Los productores no bloquean si hay espacio.
//   - wg: rastrea goroutines activas para esperar su finalización en Shutdown.
//   - quit: señal de cierre. Se cierra cuando Shutdown es llamado.
//
// Por qué un canal quit separado en lugar de cerrar jobs directamente:
// cerrar un canal desde el que también se envía requiere sincronización
// extra para evitar un panic de "send on closed channel". El canal quit
// permite señalizar sin tocar el canal de producción.
type Pool struct {
	jobs    chan ScoreEvent
	wg      sync.WaitGroup
	quit    chan struct{}
	once    sync.Once
}

// New crea un Pool con numWorkers goroutines y un buffer de bufferSize trabajos.
func New(numWorkers, bufferSize int) *Pool {
	return &Pool{
		jobs: make(chan ScoreEvent, bufferSize),
		quit: make(chan struct{}),
	}
}

// Start lanza las goroutines workers. Debe llamarse una sola vez.
// Cada worker ejecuta processor en un bucle hasta que el canal jobs
// se cierre o llegue la señal quit.
func (p *Pool) Start(numWorkers int, processor Processor) {
	for range numWorkers {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case job, ok := <-p.jobs:
					if !ok {
						return
					}
					if err := processor(job); err != nil {
						log.Printf("worker: error processing event game=%s player=%s: %v",
							job.GameID, job.PlayerID, err)
					}
				case <-p.quit:
					return
				}
			}
		}()
	}
}

// Submit envía un evento al pool sin bloquear.
// Si el canal está lleno, devuelve ErrPoolFull inmediatamente.
// El caller decide si loguear, métricas o simplemente ignorar.
func (p *Pool) Submit(event ScoreEvent) error {
	select {
	case p.jobs <- event:
		return nil
	default:
		return ErrPoolFull
	}
}

// Shutdown señaliza a los workers que paren y espera a que terminen
// o a que expire el contexto, lo que ocurra primero.
//
// Secuencia:
//  1. Cierra el canal quit (señal a todos los workers).
//  2. Espera en una goroutine separada a que wg llegue a cero.
//  3. Si el contexto expira antes, devuelve ctx.Err().
//
// El servidor HTTP debe llamar a Shutdown después de http.Server.Shutdown
// para garantizar que no entran nuevos trabajos mientras drenan los actuales.
func (p *Pool) Shutdown(ctx context.Context) error {
	p.once.Do(func() {
		close(p.quit)
	})

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
