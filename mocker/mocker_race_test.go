package mocker_test

import (
	"sync"
	"testing"

	"github.com/lfsc09/k-test-n-stress/mocker"
	"github.com/stretchr/testify/assert"
)

// TestConcurrentGenerate spawns 8 goroutines, each with its own mocker.New()
// instance, and concurrently calls Generate for functions that previously used
// the global math/rand source. Running with "go test -race ./mocker/..." must
// report no data races.
func TestConcurrentGenerate(t *testing.T) {
	const numGoroutines = 8

	type call struct {
		fn     string
		params []string
	}

	calls := []call{
		{fn: "Person.cpf", params: nil},
		{fn: "Company.cnpj", params: nil},
		{fn: "Payment.creditCardCvv", params: nil},
		{fn: "Regex.regex", params: []string{"/[0-9]{4}/"}},
		{fn: "Date.date", params: nil},
		{fn: "Date.time", params: nil},
		{fn: "Date.datetime", params: nil},
	}

	var wg sync.WaitGroup
	for range numGoroutines {
		wg.Go(func() {
			m := mocker.New()
			for _, c := range calls {
				result, err := m.Generate(c.fn, c.params)
				assert.NoError(t, err, "Generate(%q) returned error", c.fn)
				assert.NotEmpty(t, result, "Generate(%q) returned empty string", c.fn)
			}
		})
	}
	wg.Wait()
}
