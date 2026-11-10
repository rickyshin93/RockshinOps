package main

import (
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
)

var CalculatorOpened = mygo.NewEvent[string]("calculatorOpened")
var CalculatorHistoryChanged = mygo.NewEvent[bool]("calculatorHistoryChanged")

type CalculationEntry struct {
	Expression string           `json:"expression"`
	Result     CalculationValue `json:"result"`
	CreatedAt  time.Time        `json:"createdAt"`
	Error      string           `json:"error,omitempty"`
	Pending    bool             `json:"pending,omitempty"`
}

var errCalculationNotFound = errors.New("calculation_not_found")
var errCalculationHistoryChanged = errors.New("calculation_history_changed")

type CalculatorFormat struct {
	Grouping      string `json:"grouping"`
	DecimalPlaces int    `json:"decimalPlaces"`
}

const maxCalculatorDecimalPlaces = 10

func (f CalculatorFormat) validate() error {
	switch f.Grouping {
	case "auto", "myriads", "thousands", "none":
	default:
		return errors.New("unsupported number grouping")
	}
	if f.DecimalPlaces < 0 || f.DecimalPlaces > maxCalculatorDecimalPlaces {
		return errors.New("decimal places must be between 0 and 10")
	}
	return nil
}

type CalculatorSheet struct {
	Format           CalculatorFormat   `json:"format"`
	History          []CalculationEntry `json:"history"`
	MaxDecimalPlaces int                `json:"maxDecimalPlaces"`
}

type CalculatorSaveResult struct {
	Entry CalculationEntry `json:"entry"`
	Sheet CalculatorSheet  `json:"sheet"`
}

func (s *System) calculatorFormatPath() string {
	return filepath.Join(filepath.Dir(s.preferencesPath), "calculator-format.json")
}

func (s *System) readCalculatorFormat() (CalculatorFormat, error) {
	data, err := os.ReadFile(s.calculatorFormatPath())
	if errors.Is(err, os.ErrNotExist) {
		return s.config.CalculatorFormat, nil
	}
	if err != nil {
		return CalculatorFormat{}, err
	}
	var format CalculatorFormat
	if err := json.Unmarshal(data, &format); err != nil {
		return CalculatorFormat{}, err
	}
	return format, format.validate()
}

func (s *System) SaveCalculatorFormat(format CalculatorFormat) error {
	if err := format.validate(); err != nil {
		return err
	}
	unlock, err := s.lockCalculatorWrite()
	if err != nil {
		return err
	}
	defer unlock()
	return writeCalculatorJSON(s.calculatorFormatPath(), format)
}

func (s *System) readCalculatorInputs() (CalculatorFormat, []CalculationEntry, error) {
	s.calculatorMu.Lock()
	defer s.calculatorMu.Unlock()
	format, err := s.readCalculatorFormat()
	if err != nil {
		return CalculatorFormat{}, nil, err
	}
	history, err := s.readCalculatorHistory()
	return format, history, err
}

func (s *System) CalculatorSheet() (CalculatorSheet, error) {
	format, history, err := s.readCalculatorInputs()
	if err != nil {
		return CalculatorSheet{}, err
	}
	return calculatorSheet(format, history), nil
}

func calculatorSheet(format CalculatorFormat, history []CalculationEntry) CalculatorSheet {
	if format.DecimalPlaces != calculationPlaces {
		for i, value := range calculationReferences(history) {
			if value != nil {
				history[len(history)-1-i].Result = displayCalculationAt(value, format.DecimalPlaces)
			}
		}
	}
	return CalculatorSheet{Format: format, History: history, MaxDecimalPlaces: maxCalculatorDecimalPlaces}
}

func (s *System) CalculatorPreview(expression string, before *time.Time) (*CalculationValue, error) {
	result, err := s.resolveCalculation(expression, before)
	switch err {
	case errInvalidCalculation, errCalculationTooLong, errDivisionByZero, errCalculationTooLarge,
		errCalculationReference, errCalculationDependency, errCalculationNotFound:
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CalculatorPreviewEdit returns the edited row and all newer rows, newest first.
// Invalid drafts have no result and invalidate only rows that depend on them.
func (s *System) CalculatorPreviewEdit(createdAt time.Time, expression string, bindings ...map[string]string) ([]CalculationEntry, error) {
	format, history, err := s.readCalculatorInputs()
	if err != nil {
		return nil, err
	}
	index := calculationIndex(history, createdAt)
	if index < 0 {
		return nil, errCalculationNotFound
	}
	if err := validateCalculationBindings(history, bindings); err != nil {
		return nil, err
	}
	history[index].Expression = strings.TrimSpace(expression)
	preview := history[:index+1]
	recalculateCalculatorRows(preview, calculationReferences(history[index+1:]), format.DecimalPlaces)
	return preview, nil
}

func (s *System) resolveCalculation(expression string, before *time.Time) (CalculationValue, error) {
	expression = strings.TrimSpace(expression)
	s.calculatorMu.Lock()
	defer s.calculatorMu.Unlock()
	format, err := s.readCalculatorFormat()
	if err != nil {
		return CalculationValue{}, err
	}
	var prior []*big.Rat
	if strings.ContainsRune(expression, '#') {
		history, err := s.readCalculatorHistory()
		if err != nil {
			return CalculationValue{}, err
		}
		if before != nil {
			index := calculationIndex(history, *before)
			if index < 0 {
				return CalculationValue{}, errCalculationNotFound
			}
			history = history[index+1:]
		}
		prior = calculationReferences(history)
	}
	exact, err := parseCalculation(expression, false, prior)
	if err != nil {
		return CalculationValue{}, err
	}
	return displayCalculationAt(exact, format.DecimalPlaces), nil
}

func (s *System) calculatorPath() string {
	return filepath.Join(filepath.Dir(s.preferencesPath), "calculator.json")
}

func (s *System) lockCalculatorWrite() (func(), error) {
	s.calculatorMu.Lock()
	dir := filepath.Dir(s.preferencesPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		s.calculatorMu.Unlock()
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(dir, "calculator.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		s.calculatorMu.Unlock()
		return nil, err
	}
	if err := lockCalculatorFile(file); err != nil {
		_ = file.Close()
		s.calculatorMu.Unlock()
		return nil, err
	}
	return func() {
		// Closing the handle releases the operating-system lock, including on process exit.
		_ = file.Close()
		s.calculatorMu.Unlock()
	}, nil
}

func (s *System) readCalculatorHistory() ([]CalculationEntry, error) {
	data, err := os.ReadFile(s.calculatorPath())
	if errors.Is(err, os.ErrNotExist) {
		return []CalculationEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var history []CalculationEntry
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}
	if history == nil {
		return nil, errors.New("calculator history must be an array")
	}
	seen := make(map[time.Time]bool, len(history))
	for _, entry := range history {
		id := entry.CreatedAt.UTC()
		if id.IsZero() || seen[id] {
			return nil, errors.New("calculator history contains a missing or duplicate row identity")
		}
		seen[id] = true
	}
	return history, nil
}

func writeCalculatorJSON(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(path+".tmp", append(data, '\n'), 0600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

func (s *System) CalculatorHistory() ([]CalculationEntry, error) {
	s.calculatorMu.Lock()
	defer s.calculatorMu.Unlock()
	return s.readCalculatorHistory()
}

func (s *System) CalculatorCommit(expression string) (CalculationEntry, error) {
	saved, err := s.saveCalculation(expression, time.Time{}, nil, true)
	return saved.entry, err
}

func (s *System) CalculatorUpdate(createdAt time.Time, expression string) (CalculationEntry, error) {
	if createdAt.IsZero() {
		return CalculationEntry{}, errCalculationNotFound
	}
	saved, err := s.saveCalculation(expression, createdAt, nil, false)
	return saved.entry, err
}

func (s *System) CalculatorSave(expression string, row *time.Time, bindings ...map[string]string) (CalculatorSaveResult, error) {
	return s.calculatorSave(expression, row, nil, bindings...)
}

func (s *System) CalculatorSaveWithAnswer(expression string, row *time.Time, answer *mygo.Channel[CalculationValue], bindings ...map[string]string) (CalculatorSaveResult, error) {
	return s.calculatorSave(expression, row, func(value CalculationValue) {
		if err := answer.Send(value); err != nil && !errors.Is(err, mygo.ErrChannelClosed) {
			log.Printf("send calculator answer: %v", err)
		}
	}, bindings...)
}

func (s *System) calculatorSave(expression string, row *time.Time, showAnswer func(CalculationValue), bindings ...map[string]string) (CalculatorSaveResult, error) {
	var createdAt time.Time
	if row != nil {
		if row.IsZero() {
			return CalculatorSaveResult{}, errCalculationNotFound
		}
		createdAt = *row
	}
	saved, err := s.saveCalculation(expression, createdAt, showAnswer, false, bindings...)
	if err != nil {
		return CalculatorSaveResult{}, err
	}
	return CalculatorSaveResult{Entry: saved.entry, Sheet: calculatorSheet(saved.format, saved.history)}, nil
}

func (s *System) CalculatorDeleteResult(createdAt time.Time) (CalculatorSheet, error) {
	return s.calculatorDeleteResult(createdAt, false)
}

func (s *System) CalculatorDeleteEquality(createdAt time.Time, displayed ...string) (CalculatorSheet, error) {
	return s.calculatorDeleteResult(createdAt, true, displayed...)
}

func (s *System) calculatorDeleteResult(createdAt time.Time, keepAnswer bool, displayed ...string) (CalculatorSheet, error) {
	unlock, err := s.lockCalculatorWrite()
	if err != nil {
		return CalculatorSheet{}, err
	}
	defer unlock()
	format, err := s.readCalculatorFormat()
	if err != nil {
		return CalculatorSheet{}, err
	}
	history, err := s.readCalculatorHistory()
	if err != nil {
		return CalculatorSheet{}, err
	}
	index := calculationIndex(history, createdAt)
	if index < 0 {
		return CalculatorSheet{}, errCalculationNotFound
	}
	entry := &history[index]
	prior := calculationReferences(history[index+1:])
	if keepAnswer {
		if entry.Pending || entry.Error != "" {
			return CalculatorSheet{}, errCalculationDependency
		}
		exact, err := parseCalculation(entry.Expression, false, prior)
		if err != nil {
			return CalculatorSheet{}, err
		}
		answer := displayCalculationAt(exact, format.DecimalPlaces).Value
		if len(displayed) > 0 && displayed[0] != answer {
			return CalculatorSheet{}, errCalculationHistoryChanged
		}
		entry.Expression += answer
	}
	entry.Pending = true
	entry.Result = CalculationValue{}
	entry.Error = ""
	recalculateCalculatorRows(history[:index], append(prior, nil), calculationPlaces)
	if err := writeCalculatorJSON(s.calculatorPath(), history); err != nil {
		return CalculatorSheet{}, err
	}
	return calculatorSheet(format, history), nil
}

func (s *System) CalculatorDeleteRow(createdAt time.Time) (CalculatorSheet, error) {
	unlock, err := s.lockCalculatorWrite()
	if err != nil {
		return CalculatorSheet{}, err
	}
	defer unlock()
	format, err := s.readCalculatorFormat()
	if err != nil {
		return CalculatorSheet{}, err
	}
	history, err := s.readCalculatorHistory()
	if err != nil {
		return CalculatorSheet{}, err
	}
	index := calculationIndex(history, createdAt)
	if index < 0 {
		return CalculatorSheet{}, errCalculationNotFound
	}
	deletedRow := len(history) - index
	for i := range index {
		history[i].Expression = calculatorReferenceToken.ReplaceAllStringFunc(history[i].Expression, func(token string) string {
			row, err := strconv.Atoi(token[1:])
			if err != nil || row < deletedRow {
				return token
			}
			// A removed source keeps an unavailable reference without pointing at its successor.
			if row == deletedRow {
				return "#0"
			}
			return "#" + strconv.Itoa(row-1)
		})
	}
	history = slices.Delete(history, index, index+1)
	recalculateCalculatorRows(history[:index], calculationReferences(history[index:]), calculationPlaces)
	if err := writeCalculatorJSON(s.calculatorPath(), history); err != nil {
		return CalculatorSheet{}, err
	}
	return calculatorSheet(format, history), nil
}

type calculatorSaved struct {
	entry   CalculationEntry
	format  CalculatorFormat
	history []CalculationEntry
}

func (s *System) saveCalculation(expression string, createdAt time.Time, showAnswer func(CalculationValue), reuseLast bool, bindings ...map[string]string) (calculatorSaved, error) {
	expression = strings.TrimSpace(expression)
	linked := strings.ContainsRune(expression, '#')
	var exact *big.Rat
	if showAnswer != nil && !linked {
		// Unlinked arithmetic can return an answer while another writer owns the store.
		format, err := s.readCalculatorFormat()
		if err != nil {
			return calculatorSaved{}, err
		}
		exact, err = parseCalculation(expression, false, nil)
		if err != nil {
			return calculatorSaved{}, err
		}
		showAnswer(displayCalculationAt(exact, format.DecimalPlaces))
	}
	unlock, err := s.lockCalculatorWrite()
	if err != nil {
		return calculatorSaved{}, err
	}
	defer unlock()
	format, err := s.readCalculatorFormat()
	if err != nil {
		return calculatorSaved{}, err
	}
	history, err := s.readCalculatorHistory()
	if err != nil {
		return calculatorSaved{}, err
	}
	index := -1
	if !createdAt.IsZero() {
		index = calculationIndex(history, createdAt)
		if index < 0 {
			return calculatorSaved{}, errCalculationNotFound
		}
	}
	var prior []*big.Rat
	if linked {
		if err := validateCalculationBindings(history, bindings); err != nil {
			return calculatorSaved{}, err
		}
		prior = calculationReferences(history[index+1:])
	}
	if exact == nil {
		exact, err = parseCalculation(expression, false, prior)
		if err != nil {
			return calculatorSaved{}, err
		}
		if showAnswer != nil {
			showAnswer(displayCalculationAt(exact, format.DecimalPlaces))
		}
	}
	result := displayCalculationAt(exact, format.DecimalPlaces)
	entry := CalculationEntry{Expression: expression, Result: displayCalculation(exact), CreatedAt: createdAt}
	if createdAt.IsZero() {
		if reuseLast && len(history) > 0 && history[0].Expression == expression && history[0].Result == entry.Result {
			entry := history[0]
			entry.Result = result
			return calculatorSaved{entry: entry, format: format, history: history}, nil
		}
		entry.CreatedAt = time.Now().UTC()
		for calculationIndex(history, entry.CreatedAt) >= 0 {
			entry.CreatedAt = entry.CreatedAt.Add(time.Nanosecond)
		}
		history = append([]CalculationEntry{entry}, history...)
	} else {
		history[index] = entry
		if index > 0 {
			if !linked {
				prior = calculationReferences(history[index+1:])
			}
			recalculateCalculatorRows(history[:index], append(prior, exact), calculationPlaces)
		}
	}
	if err := writeCalculatorJSON(s.calculatorPath(), history); err != nil {
		return calculatorSaved{}, err
	}
	entry.Result = result
	return calculatorSaved{entry: entry, format: format, history: history}, nil
}

func recalculateCalculatorRows(rows []CalculationEntry, prior []*big.Rat, decimals int) {
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].Pending {
			prior = append(prior, nil)
			continue
		}
		exact, err := parseCalculation(rows[i].Expression, false, prior)
		prior = append(prior, exact)
		if err != nil {
			rows[i].Error = err.Error()
			rows[i].Result = CalculationValue{}
		} else {
			rows[i].Error = ""
			rows[i].Result = displayCalculationAt(exact, decimals)
		}
	}
}

func calculationIndex(history []CalculationEntry, createdAt time.Time) int {
	return slices.IndexFunc(history, func(entry CalculationEntry) bool { return entry.CreatedAt.Equal(createdAt) })
}

// Ordinals in the editor refer to the rows it displayed. A different process
// may have deleted a row since then, even when all writes take the file lock.
func validateCalculationBindings(history []CalculationEntry, bindings []map[string]string) error {
	for _, binding := range bindings {
		for token, expected := range binding {
			number, err := strconv.Atoi(strings.TrimPrefix(token, "#"))
			actual := ""
			if err == nil && number > 0 && number <= len(history) {
				actual = history[len(history)-number].CreatedAt.Format(time.RFC3339Nano)
			}
			if actual != expected {
				return errCalculationHistoryChanged
			}
		}
	}
	return nil
}

// History is stored newest first and shown oldest first. References point to
// preceding rows, whose numbers are remapped on deletion.
func calculationReferences(history []CalculationEntry) []*big.Rat {
	prior := make([]*big.Rat, 0, len(history))
	for i := len(history) - 1; i >= 0; i-- {
		var exact *big.Rat
		if !history[i].Pending {
			exact, _ = parseCalculation(history[i].Expression, false, prior)
		}
		prior = append(prior, exact)
	}
	return prior
}

func (s *System) CalculatorClearHistory() error {
	unlock, err := s.lockCalculatorWrite()
	if err != nil {
		return err
	}
	defer unlock()
	return writeCalculatorJSON(s.calculatorPath(), []CalculationEntry{})
}

func notifyCalculatorHistory() {
	if err := CalculatorHistoryChanged.Broadcast(true); err != nil {
		log.Printf("calculator history update: %v", err)
	}
}

func (s *System) CalculatorCopy(value string) error {
	digits := strings.TrimPrefix(value, "-")
	if digits == "" || calculatorNumber.FindString(digits) != digits {
		return errInvalidCalculation
	}
	return copyCalculation(value)
}

func copyCalculation(value string) error {
	return mygo.Clipboard.Write(transfer.TextData(value))
}
