package worker

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

// ErrPoolFull se devuelve cuando el canal de trabajos está lleno.
// Esto es backpressure deliberada: el handler HTTP nunca bloquea esperando
// que el pool tenga espacio. Si el pool está saturado, el efecto secundario
// se descarta y el cliente recibe igualmente su respuesta 201.
var ErrPoolFull = errors.New("worker pool is full")

// Processor es la función que ejecuta cada worker al recibir un evento.
type Processor func(ScoreEvent) error

// ErrorHandler es un callback opcional invocado cuando un evento falla
// todos sus intentos. Permite al caller decidir si loguear, enviar a
// una dead-letter queue, emitir una métrica, etc.
type ErrorHandler func(event ScoreEvent, err error, attempts int)

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
	jobs       chan ScoreEvent
	wg         sync.WaitGroup
	quit       chan struct{}
	once       sync.Once
	maxRetries int
	retryDelay time.Duration
	onError    ErrorHandler
}

// Option permite configurar el Pool con opciones opcionales.
type Option func(*Pool)

// WithRetry configura el número máximo de reintentos y la espera entre ellos.
// Por defecto no hay reintentos (maxRetries=0).
func WithRetry(maxRetries int, delay time.Duration) Option {
	return func(p *Pool) {
		p.maxRetries = maxRetries
		p.retryDelay = delay
	}
}

// WithErrorHandler registra un callback para eventos que fallan todos los reintentos.
func WithErrorHandler(h ErrorHandler) Option {
	return func(p *Pool) {
		p.onError = h
	}
}

// New crea un Pool con el buffer dado y las opciones proporcionadas.
func New(bufferSize int, opts ...Option) *Pool {
	p := &Pool{
		jobs:       make(chan ScoreEvent, bufferSize),
		quit:       make(chan struct{}),
		retryDelay: 100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Start lanza numWorkers goroutines. Debe llamarse una sola vez.
// Cada worker ejecuta processor con reintentos hasta maxRetries veces.
//
// Por qué no usamos recover() para capturar panics en el processor:
// un panic en una goroutine que no se recupera mata el proceso entero.
// El processor no debería panic — si lo hace, es un bug que queremos
// ver en los logs, no silenciar. La decisión de recuperar o no depende
// del contexto; aquí preferimos fallar ruidosamente.
//
// Comportamiento en shutdown:
// Cuando se cierra quit, el worker termina el job actual y luego drena
// el buffer con lecturas no-bloqueantes antes de salir. Esto garantiza
// que los jobs ya encolados se procesan aunque llegue la señal de cierre.
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
					p.process(job, processor)
				case <-p.quit:
					// Drenar el buffer antes de salir: cualquier job que ya
					// estaba encolado cuando llegó la señal se procesa igualmente.
					for {
						select {
						case job, ok := <-p.jobs:
							if !ok {
								return
							}
							p.process(job, processor)
						default:
							return
						}
					}
				}
			}
		}()
	}
}

// process ejecuta el processor con reintentos y llama al ErrorHandler si falla.
func (p *Pool) process(job ScoreEvent, processor Processor) {
	var lastErr error
	for attempt := range p.maxRetries + 1 {
		lastErr = processor(job)
		if lastErr == nil {
			return
		}
		if attempt < p.maxRetries {
			log.Printf("worker: retry %d/%d game=%s player=%s err=%v",
				attempt+1, p.maxRetries, job.GameID, job.PlayerID, lastErr)
			time.Sleep(p.retryDelay)
		}
	}

	// Todos los intentos fallaron
	if p.onError != nil {
		p.onError(job, lastErr, p.maxRetries+1)
	} else {
		log.Printf("worker: event dropped after %d attempts game=%s player=%s err=%v",
			p.maxRetries+1, job.GameID, job.PlayerID, lastErr)
	}
}

// Submit envía un evento al pool sin bloquear.
// Si el canal está lleno, devuelve ErrPoolFull inmediatamente.
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
// Secuencia correcta desde main:
//  1. http.Server.Shutdown → deja de aceptar requests nuevos.
//  2. pool.Shutdown → drena los jobs en vuelo.
//  3. db.Close (defer) → cierra Postgres.
//
// Invertir el orden 2 y 3 causaría errores en workers que aún usan la DB.
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
