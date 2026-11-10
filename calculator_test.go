package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCalculatorPrecisionAndErrors(t *testing.T) {
	for _, tc := range []struct {
		expression string
		want       CalculationValue
	}{
		{"-42", CalculationValue{Value: "-42"}},
		{"1 / 8", CalculationValue{Value: "0.125"}},
		{"(1 / 3) * 3", CalculationValue{Value: "1"}},
		{"2 / 3", CalculationValue{Value: "0.6667", Approximate: true}},
		{"-0.00005", CalculationValue{Value: "-0.0001", Approximate: true}},
		{"-0.00004", CalculationValue{Value: "0", Approximate: true}},
		{"9999.99995", CalculationValue{Value: "10000", Approximate: true}},
	} {
		if got, err := evaluateCalculation(tc.expression, false); err != nil || got != tc.want {
			t.Errorf("%q: got %+v, %v; want %+v", tc.expression, got, err, tc.want)
		}
	}
	if _, err := evaluateCalculation("1/(2-2)", false); !errors.Is(err, errDivisionByZero) {
		t.Fatalf("division by zero: %v", err)
	}
	if _, err := evaluateCalculation(strings.Repeat("(", 257), false); !errors.Is(err, errCalculationTooLong) {
		t.Fatalf("long expression: %v", err)
	}
}

func TestCalculatorHistoryPersistsConcurrentCommits(t *testing.T) {
	preferences := filepath.Join(t.TempDir(), "config", "preferences.json")
	s := calculatorTestSystem(t, preferences)
	if got, err := s.CalculatorPreview("1 / 3", nil); err != nil || got == nil || got.Value != "0.3333" {
		t.Fatalf("preview: %+v, %v", got, err)
	}
	if _, err := s.CalculatorCommit("1 / 0"); !errors.Is(err, errDivisionByZero) {
		t.Fatalf("invalid commit: %v", err)
	}
	if history, err := s.CalculatorHistory(); err != nil || len(history) != 0 {
		t.Fatalf("preview or invalid commit changed history: %+v, %v", history, err)
	}
	var wg sync.WaitGroup
	for _, expression := range []string{"11851-64", "1 / 3", "12345678+1"} {
		wg.Go(func() {
			if _, err := s.CalculatorCommit(expression); err != nil {
				t.Errorf("commit %q: %v", expression, err)
			}
		})
	}
	wg.Wait()
	reopened := calculatorTestSystem(t, preferences)
	history, err := reopened.CalculatorHistory()
	if err != nil || len(history) != 3 {
		t.Fatalf("reloaded history: %+v, %v", history, err)
	}
	if _, err := reopened.CalculatorCommit(history[0].Expression); err != nil {
		t.Fatal(err)
	}
	if got, err := reopened.CalculatorHistory(); err != nil || len(got) != 3 {
		t.Fatalf("repeating the latest calculation should reuse its entry: %+v, %v", got, err)
	}
	edited, err := reopened.CalculatorUpdate(history[1].CreatedAt, "4 / 3")
	if err != nil || edited.Result.Value != "1.3333" || !edited.CreatedAt.Equal(history[1].CreatedAt) {
		t.Fatalf("update changed the record identity or failed to calculate: %+v, %v", edited, err)
	}
	if got, err := s.CalculatorHistory(); err != nil || len(got) != 3 || got[1] != edited || got[0] != history[0] || got[2] != history[2] {
		t.Fatalf("updated record must stay in its original position: %+v, %v", got, err)
	}
	if err := reopened.CalculatorClearHistory(); err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.CalculatorUpdate(edited.CreatedAt, "1+2"); err == nil {
		t.Fatal("updating a cleared record must fail")
	}
	if history, err := s.CalculatorHistory(); err != nil || history == nil || len(history) != 0 {
		t.Fatalf("cleared history: %+v, %v", history, err)
	}
}

func TestCalculatorWritersSerializeAcrossSystems(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	first := calculatorTestSystem(t, path)
	second := calculatorTestSystem(t, path)
	if _, err := first.CalculatorCommit("1"); err != nil {
		t.Fatal(err)
	}
	entry, err := first.CalculatorCommit("#1+1")
	if err != nil {
		t.Fatal(err)
	}
	readingDone := make(chan struct{})
	release := make(chan struct{})
	editDone := make(chan error, 1)
	go func() {
		_, err := first.saveCalculation("#1+3", entry.CreatedAt, func(CalculationValue) {
			close(readingDone)
			<-release
		}, false)
		editDone <- err
	}()
	select {
	case <-readingDone:
	case err := <-editDone:
		t.Fatalf("edit did not acquire the store: %v", err)
	}
	appendDone := make(chan error, 1)
	go func() {
		_, err := second.CalculatorSave("2+3", nil)
		appendDone <- err
	}()
	select {
	case err := <-appendDone:
		close(release)
		<-editDone
		t.Fatalf("append completed while another instance was editing: %v", err)
	case <-time.After(75 * time.Millisecond):
		close(release)
	}
	if err := <-editDone; err != nil {
		t.Fatal(err)
	}
	if err := <-appendDone; err != nil {
		t.Fatal(err)
	}
	history, err := first.CalculatorHistory()
	if err != nil || len(history) != 3 || history[0].Expression != "2+3" || history[1].Expression != "#1+3" {
		t.Fatalf("both writes must survive: %+v, %v", history, err)
	}
}

func TestCalculatorUnlinkedAnswerPrecedesWaitingForWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	first := calculatorTestSystem(t, path)
	second := calculatorTestSystem(t, path)
	unlock, err := first.lockCalculatorWrite()
	if err != nil {
		t.Fatal(err)
	}
	shown := make(chan CalculationValue, 1)
	saved := make(chan error, 1)
	go func() {
		_, err := second.calculatorSave("5+7", nil, func(value CalculationValue) { shown <- value })
		saved <- err
	}()
	select {
	case value := <-shown:
		if value.Value != "12" {
			unlock()
			t.Fatalf("wrong first answer: %+v", value)
		}
	case <-time.After(time.Second):
		unlock()
		t.Fatal("answer waited for another writer")
	}
	unlock()
	if err := <-saved; err != nil {
		t.Fatal(err)
	}
}

func TestCalculatorExplicitSavesAppendIdenticalRows(t *testing.T) {
	s := calculatorTestSystem(t, filepath.Join(t.TempDir(), "preferences.json"))
	first, err := s.CalculatorSave("1+2", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.CalculatorSave("1+2", nil)
	if err != nil || len(second.Sheet.History) != 2 || second.Entry.CreatedAt.Equal(first.Entry.CreatedAt) {
		t.Fatalf("explicit repeat did not append a row: %+v, %v", second, err)
	}
}

func TestCalculatorRejectsMovedReferenceBeforeAnswer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	first, second := calculatorTestSystem(t, path), calculatorTestSystem(t, path)
	var sources []CalculationEntry
	for _, expression := range []string{"10", "20", "30"} {
		entry, err := first.CalculatorCommit(expression)
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, entry)
	}
	binding := map[string]string{"#2": sources[1].CreatedAt.Format(time.RFC3339Nano)}
	if saved, err := first.CalculatorSave("#2+1", nil, binding); err != nil || saved.Entry.Result.Value != "21" {
		t.Fatalf("same ordinal should survive appends: %+v, %v", saved.Entry, err)
	}
	if _, err := second.CalculatorDeleteRow(sources[0].CreatedAt); err != nil {
		t.Fatal(err)
	}
	_, err := first.calculatorSave("#2+1", nil, func(CalculationValue) {
		t.Error("stale reference published a different answer")
	}, binding)
	if !errors.Is(err, errCalculationHistoryChanged) {
		t.Fatalf("stale reference was not rejected: %v", err)
	}
}

func TestCalculatorHistoryRejectsMissingRowIdentity(t *testing.T) {
	s := calculatorTestSystem(t, filepath.Join(t.TempDir(), "preferences.json"))
	data := []byte("[null]\n")
	if err := os.WriteFile(s.calculatorPath(), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CalculatorSheet(); err == nil {
		t.Fatal("history exposed an entry without an identity")
	}
	if _, err := s.CalculatorSave("1+1", nil); err == nil {
		t.Fatal("save rewrote damaged history")
	}
	if got, err := os.ReadFile(s.calculatorPath()); err != nil || !slices.Equal(got, data) {
		t.Fatalf("original history changed: %q, %v", got, err)
	}
}

func TestCalculatorLinksPreservePrecisionAndRecalculate(t *testing.T) {
	s := calculatorTestSystem(t, filepath.Join(t.TempDir(), "preferences.json"))
	source, err := s.CalculatorCommit("1 / 3")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ expression, want string }{
		{"#1 * 3", "1"},
		{"#2 + #1 * 6", "3"},
		{"-#3", "-3"},
	} {
		if entry, err := s.CalculatorCommit(tc.expression); err != nil || entry.Result.Value != tc.want {
			t.Fatalf("%s: %+v, %v; want %s", tc.expression, entry, err, tc.want)
		}
	}
	if _, err := s.CalculatorUpdate(source.CreatedAt, "#2"); !errors.Is(err, errCalculationReference) {
		t.Fatalf("forward reference: %v", err)
	}
	if _, err := s.CalculatorUpdate(source.CreatedAt, "2 / 3"); err != nil {
		t.Fatal(err)
	}
	reopened := calculatorTestSystem(t, s.preferencesPath)
	history, err := reopened.CalculatorHistory()
	if err != nil || len(history) != 4 || history[0].Result.Value != "-6" ||
		history[1].Result.Value != "6" || history[2].Result.Value != "2" ||
		!history[3].CreatedAt.Equal(source.CreatedAt) {
		t.Fatalf("reopened linked history: %+v, %v", history, err)
	}
	if got, err := reopened.CalculatorPreview("#1 * 3", nil); err != nil || got == nil || got.Value != "2" {
		t.Fatalf("linked preview: %+v, %v", got, err)
	}
	if got, err := reopened.CalculatorPreview("#2", &source.CreatedAt); err != nil || got != nil {
		t.Fatalf("preview allowed a forward reference: %+v, %v", got, err)
	}
}

func TestCalculatorEditPreviewRecalculatesWithoutSaving(t *testing.T) {
	s := calculatorTestSystem(t, filepath.Join(t.TempDir(), "preferences.json"))
	for _, expression := range []string{"2+2", "1/3", "#2*3", "#3+1", "#1+4"} {
		if _, err := s.CalculatorCommit(expression); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SaveCalculatorFormat(CalculatorFormat{Grouping: "none", DecimalPlaces: 2}); err != nil {
		t.Fatal(err)
	}
	saved, err := s.CalculatorHistory()
	if err != nil || len(saved) != 5 {
		t.Fatalf("saved history = %+v, %v", saved, err)
	}
	preview, err := s.CalculatorPreviewEdit(saved[3].CreatedAt, "2/3")
	if err != nil || len(preview) != 4 || preview[3].Result.Value != "0.67" ||
		preview[2].Result.Value != "2" || preview[1].Result.Value != "3" || preview[0].Result.Value != "8" {
		t.Fatalf("edited preview = %+v, %v", preview, err)
	}
	preview, err = s.CalculatorPreviewEdit(saved[3].CreatedAt, "2/")
	if err != nil || len(preview) != 4 || preview[3].Error != errInvalidCalculation.Error() ||
		preview[2].Error != errCalculationDependency.Error() || preview[1].Error != errCalculationDependency.Error() || preview[0].Result.Value != "8" {
		t.Fatalf("incomplete preview = %+v, %v", preview, err)
	}
	after, err := s.CalculatorHistory()
	if err != nil || !slices.Equal(saved, after) {
		t.Fatalf("preview changed saved history: %+v, %v", after, err)
	}
}

func TestCalculatorSaveReturnsUpdatedSheet(t *testing.T) {
	s := calculatorTestSystem(t, filepath.Join(t.TempDir(), "preferences.json"))
	format := CalculatorFormat{Grouping: "none", DecimalPlaces: 2}
	if err := s.SaveCalculatorFormat(format); err != nil {
		t.Fatal(err)
	}
	first, err := s.CalculatorSave("1/3", nil)
	if err != nil || first.Entry.Result.Value != "0.33" || first.Sheet.Format != format ||
		!slices.Equal(first.Sheet.History, []CalculationEntry{first.Entry}) {
		t.Fatalf("initial save: %+v, %v", first, err)
	}
	if _, err := s.CalculatorSave("#1*3", nil); err != nil {
		t.Fatal(err)
	}
	updated, err := s.CalculatorSave("2/3", &first.Entry.CreatedAt)
	if err != nil || updated.Entry.Result.Value != "0.67" || len(updated.Sheet.History) != 2 ||
		updated.Sheet.History[0].Result.Value != "2" || updated.Sheet.History[1] != updated.Entry {
		t.Fatalf("edited save: %+v, %v", updated, err)
	}
	sheet, err := s.CalculatorSheet()
	if err != nil || !slices.Equal(sheet.History, updated.Sheet.History) {
		t.Fatalf("saved sheet: %+v, %v", sheet, err)
	}
}

func TestCalculatorReportsAnswerBeforeWritingHistory(t *testing.T) {
	s := calculatorTestSystem(t, filepath.Join(t.TempDir(), "preferences.json"))
	var shown []CalculationValue
	saved, err := s.saveCalculation("1/3", time.Time{}, func(value CalculationValue) {
		if _, err := os.Stat(s.calculatorPath()); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("history was already written: %v", err)
		}
		shown = append(shown, value)
	}, false)
	if err != nil || !slices.Equal(shown, []CalculationValue{saved.entry.Result}) || saved.entry.Result.Value != "0.3333" {
		t.Fatalf("shown = %+v, saved = %+v, error = %v", shown, saved.entry, err)
	}
	if _, err := s.saveCalculation("1/0", time.Time{}, func(CalculationValue) {
		t.Error("invalid expression reported an answer")
	}, false); !errors.Is(err, errDivisionByZero) {
		t.Fatalf("invalid save: %v", err)
	}
}

func TestCalculatorDeleteResultKeepsExpressionAndReferenceIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	s := calculatorTestSystem(t, path)
	source, err := s.CalculatorCommit("2*(3+4)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CalculatorCommit("#1+1"); err != nil {
		t.Fatal(err)
	}
	removed, err := s.CalculatorDeleteResult(source.CreatedAt)
	if err != nil || len(removed.History) != 2 || !removed.History[1].Pending ||
		removed.History[1].Expression != source.Expression || !removed.History[1].CreatedAt.Equal(source.CreatedAt) ||
		removed.History[0].Error != errCalculationDependency.Error() {
		t.Fatalf("removed result: %+v, %v", removed, err)
	}
	reopened := calculatorTestSystem(t, path)
	if _, err := reopened.CalculatorCommit("#1+5"); !errors.Is(err, errCalculationDependency) {
		t.Fatalf("reference to removed result = %v", err)
	}
	saved, err := reopened.CalculatorSave(source.Expression, &source.CreatedAt)
	if err != nil || saved.Entry.Pending || saved.Sheet.History[0].Result.Value != "15" || saved.Sheet.History[0].Error != "" {
		t.Fatalf("recalculated result: %+v, %v", saved, err)
	}
}

func TestCalculatorDeleteEqualityJoinsExpressionAndDisplayedAnswer(t *testing.T) {
	for _, test := range []struct {
		expression string
		joined     string
	}{
		{"1+2", "1+23"},
		{"1/3", "1/30.33"},
	} {
		t.Run(test.expression, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "preferences.json")
			s := calculatorTestSystem(t, path)
			if err := s.SaveCalculatorFormat(CalculatorFormat{Grouping: "auto", DecimalPlaces: 2}); err != nil {
				t.Fatal(err)
			}
			source, err := s.CalculatorCommit(test.expression)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.CalculatorCommit("#1+1"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.CalculatorDeleteEquality(source.CreatedAt, "99"); !errors.Is(err, errCalculationHistoryChanged) {
				t.Fatalf("equality deletion accepted a stale displayed answer: %v", err)
			}
			removed, err := s.CalculatorDeleteEquality(source.CreatedAt)
			if err != nil {
				t.Fatal(err)
			}
			want := CalculationEntry{Expression: test.joined, CreatedAt: source.CreatedAt, Pending: true}
			if len(removed.History) != 2 || removed.History[1] != want || removed.History[0].Error != errCalculationDependency.Error() {
				t.Fatalf("removed equality: %+v", removed.History)
			}
			reopened := calculatorTestSystem(t, path)
			saved, err := reopened.CalculatorSave(test.joined, &source.CreatedAt)
			if err != nil || saved.Entry.Pending || saved.Sheet.History[0].Error != "" {
				t.Fatalf("recalculated joined expression: %+v, %v", saved, err)
			}
		})
	}
}

func TestCalculatorDeleteRowRemapsReferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	s := calculatorTestSystem(t, path)
	var entries []CalculationEntry
	for _, expression := range []string{"7", "10", "#1+1", "#2+2", "#3+1"} {
		entry, err := s.CalculatorCommit(expression)
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	}
	sheet, err := s.CalculatorDeleteRow(entries[1].CreatedAt)
	if err != nil || len(sheet.History) != 4 {
		t.Fatalf("deleted row: %+v, %v", sheet, err)
	}
	want := []CalculationEntry{entries[4], entries[3], entries[2], entries[0]}
	want[0].Expression = "#2+1"
	want[1].Expression = "#0+2"
	want[1].Error = errCalculationDependency.Error()
	want[1].Result = CalculationValue{}
	if !slices.Equal(sheet.History, want) {
		t.Fatalf("remapped history = %+v, want %+v", sheet.History, want)
	}
	reopened := calculatorTestSystem(t, path)
	if saved, err := reopened.CalculatorHistory(); err != nil || !slices.Equal(saved, want) {
		t.Fatalf("reopened history = %+v, %v", saved, err)
	}
	if entry, err := reopened.CalculatorCommit("#2+3"); err != nil || entry.Result.Value != "11" {
		t.Fatalf("reference after reopening: %+v, %v", entry, err)
	}
}

func TestCalculatorLinksRecoverFromInvalidUpstreamValue(t *testing.T) {
	s := calculatorTestSystem(t, filepath.Join(t.TempDir(), "preferences.json"))
	source, err := s.CalculatorCommit("2")
	if err != nil {
		t.Fatal(err)
	}
	for _, expression := range []string{"10 / #1", "#2 + 1", "3 * 3"} {
		if _, err := s.CalculatorCommit(expression); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.CalculatorUpdate(source.CreatedAt, "0"); err != nil {
		t.Fatal(err)
	}
	history, err := s.CalculatorHistory()
	if err != nil || len(history) != 4 || history[0].Result.Value != "9" ||
		history[1].Error != errCalculationDependency.Error() || history[2].Error != errDivisionByZero.Error() {
		t.Fatalf("invalid dependent rows: %+v, %v", history, err)
	}
	if _, err := s.CalculatorUpdate(source.CreatedAt, "5"); err != nil {
		t.Fatal(err)
	}
	history, err = s.CalculatorHistory()
	if err != nil || len(history) != 4 || history[1].Error != "" || history[2].Error != "" ||
		history[1].Result.Value != "3" || history[2].Result.Value != "2" {
		t.Fatalf("recovered dependent rows: %+v, %v", history, err)
	}
}

func calculatorTestSystem(t *testing.T, preferencesPath string) System {
	t.Helper()
	config, err := parseConfig(configJSON)
	if err != nil {
		t.Fatal(err)
	}
	return System{preferencesPath: preferencesPath, config: config}
}

func TestCalculatorFormatPersistsWithoutRoundingLinkedArithmetic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	s := calculatorTestSystem(t, path)
	for _, expression := range []string{"1/3", "#1*3", "120.4"} {
		if _, err := s.CalculatorCommit(expression); err != nil {
			t.Fatal(err)
		}
	}
	format := CalculatorFormat{Grouping: "thousands", DecimalPlaces: 0}
	if err := s.SaveCalculatorFormat(format); err != nil {
		t.Fatal(err)
	}
	reopened := calculatorTestSystem(t, path)
	sheet, err := reopened.CalculatorSheet()
	if err != nil || sheet.Format != format || len(sheet.History) != 3 ||
		sheet.History[0].Result != (CalculationValue{Value: "120", Approximate: true}) ||
		sheet.History[1].Result.Value != "1" || sheet.History[2].Result.Value != "0" {
		t.Fatalf("rounded calculator sheet: %+v, %v", sheet, err)
	}
	if got, err := reopened.CalculatorPreview("#1*3", nil); err != nil || got == nil || got.Value != "1" {
		t.Fatalf("linked preview lost its exact value: %+v, %v", got, err)
	}
	reopened.launcher = &launcher{}
	search := reopened.LauncherQuery("120 + .4")
	if search.CalculatorError != "" || len(search.Items) != 1 || search.Items[0].Title != "120" || search.Items[0].Grouping != "thousands" {
		t.Fatalf("launcher format: %+v", search)
	}
	if err := reopened.SaveCalculatorFormat(CalculatorFormat{Grouping: "none", DecimalPlaces: 8}); err != nil {
		t.Fatal(err)
	}
	sheet, err = reopened.CalculatorSheet()
	if err != nil || sheet.History[2].Result.Value != "0.33333333" {
		t.Fatalf("higher precision must recover digits from the expression: %+v, %v", sheet, err)
	}
	if err := reopened.SaveCalculatorFormat(CalculatorFormat{Grouping: "invalid", DecimalPlaces: 8}); err == nil {
		t.Fatal("invalid grouping must fail")
	}
}
