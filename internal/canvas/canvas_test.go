package canvas

import "testing"

func TestNewIsBlank(t *testing.T) {
	c := New(3, 2)
	if c.Width() != 3 || c.Height() != 2 {
		t.Fatalf("size = %dx%d, want 3x2", c.Width(), c.Height())
	}
	for y := range 2 {
		for x := range 3 {
			if !c.IsEmpty(x, y) {
				t.Errorf("cell (%d,%d) = %+v, want blank", x, y, c.At(x, y))
			}
		}
	}
}

func TestNewNegativeSize(t *testing.T) {
	c := New(-1, -5)
	if c.Width() != 0 || c.Height() != 0 {
		t.Errorf("size = %dx%d, want 0x0", c.Width(), c.Height())
	}
	if got := c.String(); got != "" {
		t.Errorf("String() = %q, want empty", got)
	}
}

func TestSetAndAt(t *testing.T) {
	c := New(4, 3)
	c.Set(1, 2, '@', InkCenter)

	if got := c.At(1, 2); got != (Cell{Rune: '@', Ink: InkCenter}) {
		t.Errorf("At(1,2) = %+v", got)
	}
	if c.IsEmpty(1, 2) {
		t.Error("IsEmpty(1,2) = true after Set")
	}
}

func TestOutOfBoundsIsSafe(t *testing.T) {
	c := New(2, 2)
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {2, 0}, {0, 2}, {99, 99}} {
		c.Set(p[0], p[1], 'x', InkStem) // must not panic
		if c.In(p[0], p[1]) {
			t.Errorf("In(%d,%d) = true", p[0], p[1])
		}
		if got := c.At(p[0], p[1]); got != blank {
			t.Errorf("At(%d,%d) = %+v, want blank", p[0], p[1], got)
		}
		if c.IsEmpty(p[0], p[1]) {
			t.Errorf("IsEmpty(%d,%d) = true out of bounds", p[0], p[1])
		}
		if c.SetIfEmpty(p[0], p[1], 'x', InkStem) {
			t.Errorf("SetIfEmpty(%d,%d) wrote out of bounds", p[0], p[1])
		}
	}
	if c.Row(-1) != nil || c.Row(2) != nil {
		t.Error("Row out of bounds should be nil")
	}
}

func TestSetIfEmpty(t *testing.T) {
	c := New(2, 1)
	if !c.SetIfEmpty(0, 0, 'a', InkLeaf) {
		t.Fatal("first SetIfEmpty should write")
	}
	if c.SetIfEmpty(0, 0, 'b', InkStar) {
		t.Fatal("second SetIfEmpty should not overwrite")
	}
	if got := c.At(0, 0).Rune; got != 'a' {
		t.Errorf("rune = %q, want 'a'", got)
	}
}

func TestClear(t *testing.T) {
	c := New(2, 2)
	c.Set(0, 0, 'x', InkStem)
	c.Clear()
	if !c.IsEmpty(0, 0) {
		t.Error("Clear did not blank the canvas")
	}
}

func TestRow(t *testing.T) {
	c := New(3, 2)
	c.Set(2, 1, 'z', InkSoil)
	row := c.Row(1)
	if len(row) != 3 || row[2].Rune != 'z' {
		t.Errorf("Row(1) = %+v", row)
	}
}

func TestString(t *testing.T) {
	c := New(4, 3)
	c.Set(0, 0, 'a', InkNone)
	c.Set(2, 1, '✿', InkPetalPink)
	want := "a\n  ✿\n\n"
	if got := c.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestShade(t *testing.T) {
	c := New(3, 1)
	c.Set(0, 0, 'a', InkLeaf)
	c.Shade(0, 0)
	c.Shade(1, 0)  // blank: nothing to shade
	c.Shade(-1, 0) // out of bounds: ignored
	if !c.At(0, 0).Shadow {
		t.Error("shaded cell should be in shadow")
	}
	if !c.IsEmpty(1, 0) {
		t.Error("shading a blank cell must keep it blank")
	}
	c.Set(0, 0, 'b', InkLeaf)
	if c.At(0, 0).Shadow {
		t.Error("Set should replace the whole cell, shadow included")
	}
}
