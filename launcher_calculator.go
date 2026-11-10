package main

import (
	"errors"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

var (
	calculatorOperators      = strings.NewReplacer("×", "*", "÷", "/", "−", "-")
	calculatorNumber         = regexp.MustCompile(`^(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)`)
	calculatorReferenceToken = regexp.MustCompile(`#(?:[1-9][0-9]*|0)`)
	calculatorReference      = regexp.MustCompile("^" + calculatorReferenceToken.String())
)

type calculatorParser struct {
	rest   string
	binary bool
	err    error
	prior  []*big.Rat
}

type CalculationValue struct {
	Value       string `json:"value"`
	Approximate bool   `json:"approximate"`
}

const (
	calculationPlaces  = 4
	calculationBitSize = 4096
)

var (
	errInvalidCalculation    = errors.New("invalid_expression")
	errCalculationTooLong    = errors.New("expression_too_long")
	errDivisionByZero        = errors.New("division_by_zero")
	errCalculationTooLarge   = errors.New("result_too_large")
	errCalculationReference  = errors.New("invalid_calculation_reference")
	errCalculationDependency = errors.New("invalid_calculation_dependency")
)

func evaluateCalculation(expression string, requireOperator bool) (CalculationValue, error) {
	exact, err := parseCalculation(expression, requireOperator, nil)
	if err != nil {
		return CalculationValue{}, err
	}
	return displayCalculation(exact), nil
}

func parseCalculation(expression string, requireOperator bool, prior []*big.Rat) (*big.Rat, error) {
	// The length cap bounds recursive parsing.
	expression = calculatorOperators.Replace(expression)
	if len(expression) > 256 {
		return nil, errCalculationTooLong
	}
	p := calculatorParser{rest: expression, prior: prior}
	result := p.expression(1)
	if p.err != nil {
		return nil, p.err
	}
	if result == nil || p.rest != "" || requireOperator && !p.binary {
		return nil, errInvalidCalculation
	}
	return result, nil
}

func displayCalculation(result *big.Rat) CalculationValue {
	return displayCalculationAt(result, calculationPlaces)
}

func displayCalculationAt(result *big.Rat, decimals int) CalculationValue {
	places, exact := result.FloatPrec()
	value := result.FloatString(decimals)
	if decimals > 0 {
		value = strings.TrimRight(strings.TrimRight(value, "0"), ".")
	}
	if value == "-0" {
		value = "0"
	}
	return CalculationValue{Value: value, Approximate: !exact || places > decimals}
}

func (p *calculatorParser) expression(minPrecedence int) *big.Rat {
	left := p.operand()
	for left != nil {
		p.rest = strings.TrimSpace(p.rest)
		if p.rest == "" {
			break
		}
		op := p.rest[0]
		var precedence int
		switch op {
		case '+', '-':
			precedence = 1
		case '*', '/':
			precedence = 2
		}
		if precedence < minPrecedence {
			break
		}
		p.rest = p.rest[1:]
		p.binary = true
		right := p.expression(precedence + 1)
		if right == nil {
			return nil
		}
		switch op {
		case '+':
			left.Add(left, right)
		case '-':
			left.Sub(left, right)
		case '*':
			left.Mul(left, right)
		case '/':
			if right.Sign() == 0 {
				p.err = errDivisionByZero
				return nil
			}
			left.Quo(left, right)
		}
		// Repeatedly linking and multiplying results can grow much faster
		// than the length of a single expression.
		if max(left.Num().BitLen(), left.Denom().BitLen()) > calculationBitSize {
			p.err = errCalculationTooLarge
			return nil
		}
	}
	return left
}

func (p *calculatorParser) operand() *big.Rat {
	p.rest = strings.TrimSpace(p.rest)
	if p.rest == "" {
		return nil
	}
	switch op := p.rest[0]; op {
	case '+', '-':
		p.rest = p.rest[1:]
		value := p.operand()
		if value != nil && op == '-' {
			value.Neg(value)
		}
		return value
	case '(':
		p.rest = p.rest[1:]
		value := p.expression(1)
		if rest, ok := strings.CutPrefix(p.rest, ")"); ok {
			p.rest = rest
			return value
		}
		return nil
	case '#':
		token := calculatorReference.FindString(p.rest)
		if token == "" {
			p.err = errCalculationReference
			return nil
		}
		row, err := strconv.Atoi(token[1:])
		if err != nil || row > len(p.prior) {
			p.err = errCalculationReference
			return nil
		}
		if row == 0 {
			p.err = errCalculationDependency
			return nil
		}
		value := p.prior[row-1]
		if value == nil {
			p.err = errCalculationDependency
			return nil
		}
		p.rest = p.rest[len(token):]
		return new(big.Rat).Set(value)
	default:
		number := calculatorNumber.FindString(p.rest)
		p.rest = p.rest[len(number):]
		value, _ := new(big.Rat).SetString(number)
		return value
	}
}
