package concurrency

import (
	"context"
	"sync"
)

type WorkerPool struct {
	workers   int
	taskQueue chan func() error
	wg        sync.WaitGroup
	errChan   chan error
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &WorkerPool{
		workers:   workers,
		taskQueue: make(chan func() error, workers*2),
		errChan:   make(chan error, workers),
		ctx:       ctx,
		cancel:    cancel,
	}
	pool.start()
	return pool
}

func (p *WorkerPool) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-p.ctx.Done():
					return
				case task, ok := <-p.taskQueue:
					if !ok {
						return
					}
					if err := task(); err != nil {
						select {
						case p.errChan <- err:
						default:
						}
					}
				}
			}
		}()
	}
}

func (p *WorkerPool) Submit(task func() error) {
	select {
	case <-p.ctx.Done():
		return
	case p.taskQueue <- task:
	}
}

func (p *WorkerPool) Wait() error {
	close(p.taskQueue)
	p.wg.Wait()
	close(p.errChan)

	for err := range p.errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *WorkerPool) Shutdown() {
	p.cancel()
	p.wg.Wait()
}
