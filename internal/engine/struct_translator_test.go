// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"sync"
	"testing"
)

type testMessages struct {
	Greeting  string
	Farewell  string
	ErrEmpty  string
	BtnSubmit string
}

var testEnUS = testMessages{
	Greeting:  "Hello",
	Farewell:  "Goodbye",
	ErrEmpty:  "cannot be empty",
	BtnSubmit: "Submit",
}

var testEsES = testMessages{
	Greeting:  "Hola",
	Farewell:  "Adiós",
	ErrEmpty:  "no puede estar vacío",
	BtnSubmit: "Enviar",
}

var testJaJP = testMessages{
	Greeting:  "こんにちは",
	Farewell:  "さようなら",
	ErrEmpty:  "空にできません",
	BtnSubmit: "送信",
}

func TestStructTranslator_Get(t *testing.T) {
	st := NewStructTranslator(&testEnUS)
	msg := st.Get()
	if msg.Greeting != "Hello" {
		t.Errorf("Get().Greeting = %q, want %q", msg.Greeting, "Hello")
	}
}

func TestStructTranslator_Set(t *testing.T) {
	st := NewStructTranslator(&testEnUS)
	if st.Get().Greeting != "Hello" {
		t.Fatal("initial locale wrong")
	}

	st.Set(&testEsES)
	if st.Get().Greeting != "Hola" {
		t.Errorf("after Set, Get().Greeting = %q, want %q", st.Get().Greeting, "Hola")
	}

	st.Set(&testJaJP)
	if st.Get().Greeting != "こんにちは" {
		t.Errorf("after Set, Get().Greeting = %q, want %q", st.Get().Greeting, "こんにちは")
	}
}

func TestStructTranslator_Concurrent(t *testing.T) {
	st := NewStructTranslator(&testEnUS)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			st.Set(&testEsES)
		}()
		go func() {
			defer wg.Done()
			msg := st.Get()
			// Must be one of the valid locales, never a torn read
			if msg.Greeting != "Hello" && msg.Greeting != "Hola" {
				t.Errorf("unexpected Greeting: %q", msg.Greeting)
			}
		}()
	}
	wg.Wait()
}

func TestLocaleSet_Get(t *testing.T) {
	ls := NewLocaleSet("en-US", &testEnUS)
	ls.Add("es-ES", &testEsES)
	ls.Add("ja-JP", &testJaJP)

	if ls.Get("es-ES").Greeting != "Hola" {
		t.Error("Get(es-ES) wrong")
	}
	if ls.Get("ja-JP").Greeting != "こんにちは" {
		t.Error("Get(ja-JP) wrong")
	}
	// Unknown falls back
	if ls.Get("fr-FR").Greeting != "Hello" {
		t.Error("Get(fr-FR) should fall back to en-US")
	}
}

func TestLocaleSet_Codes(t *testing.T) {
	ls := NewLocaleSet("en-US", &testEnUS)
	ls.Add("es-ES", &testEsES)

	codes := ls.Codes()
	if len(codes) != 2 {
		t.Errorf("Codes() len = %d, want 2", len(codes))
	}
}

func TestLocaleSet_SetLocale(t *testing.T) {
	ls := NewLocaleSet("en-US", &testEnUS)
	ls.Add("es-ES", &testEsES)
	st := NewStructTranslator(&testEnUS)

	found := ls.SetLocale(st, "es-ES")
	if !found {
		t.Error("SetLocale(es-ES) should return true")
	}
	if st.Get().Greeting != "Hola" {
		t.Error("after SetLocale(es-ES), greeting should be Hola")
	}

	found = ls.SetLocale(st, "fr-FR")
	if found {
		t.Error("SetLocale(fr-FR) should return false")
	}
	if st.Get().Greeting != "Hello" {
		t.Error("after SetLocale(fr-FR), should fall back to en-US")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkStructTranslator_Get(b *testing.B) {
	st := NewStructTranslator(&testEnUS)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = st.Get().Greeting
	}
}

func BenchmarkStructTranslator_MultiField(b *testing.B) {
	st := NewStructTranslator(&testEnUS)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := st.Get()
		_ = m.Greeting
		_ = m.Farewell
		_ = m.ErrEmpty
		_ = m.BtnSubmit
	}
}

func BenchmarkStructTranslator_Set(b *testing.B) {
	st := NewStructTranslator(&testEnUS)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			st.Set(&testEnUS)
		} else {
			st.Set(&testEsES)
		}
	}
}

func BenchmarkStructTranslator_Parallel(b *testing.B) {
	st := NewStructTranslator(&testEnUS)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = st.Get().Greeting
		}
	})
}

func BenchmarkLocaleSet_SetLocale(b *testing.B) {
	ls := NewLocaleSet("en-US", &testEnUS)
	ls.Add("es-ES", &testEsES)
	ls.Add("ja-JP", &testJaJP)
	st := NewStructTranslator(&testEnUS)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ls.SetLocale(st, "es-ES")
	}
}
