package transactors

import (
	"context"
	"sync"
)

// wgKey is the key type for storing WaitGroup in context
type wgKey struct{}

// InjectWG injects the WaitGroup into the context
func InjectWG(ctx context.Context, wg *sync.WaitGroup) context.Context {
	return context.WithValue(ctx, wgKey{}, wg)
}

// ExtractWG extracts the WaitGroup from the context
func ExtractWG(ctx context.Context) *sync.WaitGroup {
	if wg, ok := ctx.Value(wgKey{}).(*sync.WaitGroup); ok {
		return wg
	}
	return nil
}

// HelperExtractWG extracts the WaitGroup from context, or returns a new one if not found
func HelperExtractWG(ctx context.Context) *sync.WaitGroup {
	wg := ExtractWG(ctx)
	if wg == nil {
		wg = &sync.WaitGroup{}
	}
	return wg
}

// WithWaitGroup creates a new context with a new WaitGroup and returns both
func WithWaitGroup(ctx context.Context) (context.Context, *sync.WaitGroup) {
	wg := &sync.WaitGroup{}
	return InjectWG(ctx, wg), wg
}

// WaitAndClose runs the given function with a WaitGroup from context,
// waits for all goroutines to complete, then calls the optional close function
func WaitAndClose(ctx context.Context, fn func(), close func()) {
	wg := HelperExtractWG(ctx)
	wg.Add(1)
	defer func() {
		wg.Done()
		wg.Wait()
		if close != nil {
			close()
		}
	}()
	fn()
}
