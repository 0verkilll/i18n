// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package sample

// This file exercises all 9 translation call patterns for integration testing.
// It is NOT compiled as part of the project; it is only read by the extractor.

type translator interface {
	Translate(string) string
	TranslateWithArgs(string, ...interface{}) string
	TranslatePlural(string, int) string
	TranslateGender(string, string) string
	T(string) string
	TF(string, ...interface{}) string
	TD(string, string) string
	Has(string) bool
	HasKey(string) bool
	Key(string) string
}

type Messages struct {
	Greeting string
	Farewell string
}

type holder struct{}

func (h holder) Get() *Messages { return &Messages{} }

func calls(t translator) {
	// 1. Translate
	t.Translate("greeting")

	// 2. TranslateWithArgs
	t.TranslateWithArgs("welcome_user", "Alice")

	// 3. TranslatePlural
	t.TranslatePlural("item_count", 5)

	// 4. TranslateGender
	t.TranslateGender("salutation", "female")

	// 5. T (alias)
	t.T("farewell")

	// 6. TF (alias with args)
	t.TF("items_format", 3, "books")

	// 7. TD (with default value)
	t.TD("error.required", "This field is required")

	// 8. Has
	t.Has("optional_feature")

	// 9. Key
	t.Key("nav.home")

	// Non-literal arguments -- must be silently skipped.
	dynamicKey := "dynamic"
	t.Translate(dynamicKey)
	t.T(getKey())
}

func getKey() string { return "x" }

// Struct field patterns: Msg.Get().FieldName
func structFields() {
	var Msg holder
	_ = Msg.Get().Greeting
	_ = Msg.Get().Farewell
}
