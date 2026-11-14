package styleeq

import (
	"bytes"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/internal/apidoc"
)

// The committed file is what the generator writes from today's struct. This is
// the check that makes a hand-forgotten field impossible: add one to
// core.Style without regenerating and this fails, naming the command.
func TestTheGeneratedEqualIsUpToDate(t *testing.T) {
	root, err := apidoc.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "core")
	want, err := Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, GeneratedFile))
	if err != nil {
		t.Fatalf("core/%s is missing — run `go run ./internal/styleeq/gen`", GeneratedFile)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("core/%s is stale — run `go run ./internal/styleeq/gen`", GeneratedFile)
	}
}

// Style.Equal agrees with reflect.DeepEqual, which is what reconcile used
// before it and what the generated method claims to compute.
//
// Each round builds a random style, copies it, and changes zero, one or a few
// fields of the copy, so the pairs are equal often enough for both answers to
// be exercised. Values come from small pools (two strings, two numbers) so a
// "change" can also write back the same value, which must compare equal.
// Nested styles (HoverStyle, FocusStyle, PseudoStates) are filled to a depth of
// two, with nil and empty maps both in play.
//
// NaN is left out on purpose: DeepEqual calls a map equal to itself without
// looking inside, so a NaN in the same PseudoStates map is "equal" there and
// unequal here. No pass can carry a NaN anyway; it does not encode (render's
// unencodable).
func TestEqualAgreesWithDeepEqual(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	var equal, differ int
	for range 20000 {
		a := randomStyle(r, 2)
		b := cloneStyle(a)
		for range r.IntN(4) {
			mutate(r, b, 2)
		}
		want := reflect.DeepEqual(a, b)
		if got := a.Equal(b); got != want {
			t.Fatalf("Equal says %v, DeepEqual says %v\na: %+v\nb: %+v", got, want, a, b)
		}
		if got := b.Equal(a); got != want {
			t.Fatalf("Equal is not symmetric: b.Equal(a) = %v, want %v", got, want)
		}
		if want {
			equal++
		} else {
			differ++
		}
	}
	if equal < 1000 || differ < 1000 {
		t.Fatalf("the fuzz is lopsided (%d equal, %d differ); it is not testing both answers", equal, differ)
	}

	var nilStyle *core.Style
	if !nilStyle.Equal(nil) || nilStyle.Equal(&core.Style{}) || (&core.Style{}).Equal(nil) {
		t.Fatal("nil handling disagrees with DeepEqual")
	}
}

func randomStyle(r *rand.Rand, depth int) *core.Style {
	s := &core.Style{}
	for range 6 {
		mutate(r, s, depth)
	}
	return s
}

// mutate sets one random field of s to a random value from a small pool.
func mutate(r *rand.Rand, s *core.Style, depth int) {
	v := reflect.ValueOf(s).Elem()
	f := v.Field(r.IntN(v.NumField()))
	setRandom(r, f, depth)
}

func setRandom(r *rand.Rand, f reflect.Value, depth int) {
	switch f.Kind() {
	case reflect.String:
		f.SetString([]string{"", "a", "b"}[r.IntN(3)])
	case reflect.Bool:
		f.SetBool(r.IntN(2) == 0)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f.SetInt(int64(r.IntN(3)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		f.SetUint(uint64(r.IntN(3)))
	case reflect.Float32, reflect.Float64:
		f.SetFloat([]float64{0, 1.5, 2}[r.IntN(3)])
	case reflect.Struct:
		setRandom(r, f.Field(r.IntN(f.NumField())), depth)
	case reflect.Pointer:
		if depth == 0 || r.IntN(3) == 0 {
			f.SetZero()
			return
		}
		f.Set(reflect.ValueOf(randomStyle(r, depth-1)))
	case reflect.Map:
		switch {
		case depth == 0 || r.IntN(3) == 0:
			f.SetZero()
		case r.IntN(3) == 0:
			f.Set(reflect.MakeMap(f.Type())) // empty, not nil
		default:
			m := reflect.MakeMap(f.Type())
			for range 1 + r.IntN(2) {
				key := []string{":hover", ":focus"}[r.IntN(2)]
				m.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(*randomStyle(r, depth-1)))
			}
			f.Set(m)
		}
	default:
		// A field of a kind this test cannot fill stays zero; the generator's
		// DeepEqual fallback covers it.
	}
}

// cloneStyle is a deep copy, so mutating the copy never reaches the original
// through a shared pointer or map.
func cloneStyle(s *core.Style) *core.Style {
	if s == nil {
		return nil
	}
	c := *s
	c.HoverStyle = cloneStyle(s.HoverStyle)
	c.FocusStyle = cloneStyle(s.FocusStyle)
	if s.PseudoStates != nil {
		c.PseudoStates = make(map[string]core.Style, len(s.PseudoStates))
		for k, v := range s.PseudoStates {
			c.PseudoStates[k] = *cloneStyle(&v)
		}
	}
	return &c
}
