package main

import (
	"testing"
)

func TestWalkResult(t *testing.T) {
	w := W(0, 5)
	if res, text := walkResult(w); res != CHECK_WEATHER {
		t.Error(text, res)
	}

	w = W(0, 0)
	if res, text := walkResult(w); res != CHECK_WEATHER {
		t.Error(text, res)
	}

	w = W(-1, 5)
	if res, text := walkResult(w); res != GOOD_WEATHER {
		t.Error(text, res)
	}

	w = W(-14, 7)
	if res, text := walkResult(w); res != GOOD_WEATHER {
		t.Error(text, res)
	}

	w = W(-15, 7.2)
	if res, text := walkResult(w); res != BAD_WEATHER {
		t.Error(text, res)
	}

	w = W(-15, 7)
	if res, text := walkResult(w); res != GOOD_WEATHER {
		t.Error(text, res)
	}

	w = W(-15, 6.9)
	if res, text := walkResult(w); res != GOOD_WEATHER {
		t.Error(text, res)
	}

	w = W(-17, 7)
	if res, text := walkResult(w); res != GOOD_WEATHER {
		t.Error(text, res)
	}

	w = W(-30, 6)
	if res, text := walkResult(w); res != GOOD_WEATHER {
		t.Error(text, res)
	}

	w = W(-31, 6)
	if res, text := walkResult(w); res != BAD_WEATHER {
		t.Error(text, res)
	}
}

func W(t float32, v float32) *Weather {
	return &Weather{
		Main: Main{
			Temp: t,
		},
		Wind: Wind{
			Speed: v,
		},
	}
}
