package canvasfixture

import "testing"

// The table is not vacuous: every case draws something, and the two ending
// cases stop where they should.
func TestCasesDrawAndStop(t *testing.T) {
	stops := map[string]int{"truncated operation": 1, "unknown opcode": 1}
	for _, c := range Cases() {
		w := WantFor(c)
		if len(w.Calls) == 0 {
			t.Errorf("%s draws nothing", c.Name)
		}
		if n, ok := stops[c.Name]; ok && len(w.Calls) != n {
			t.Errorf("%s: %d calls, want %d", c.Name, len(w.Calls), n)
		}
	}
}
