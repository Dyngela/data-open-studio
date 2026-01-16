package gen

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"api/internal/api/models"
)

// ExecuteMap executes a Map node with transformation logic
// For single output, returns a single stream
// For multiple outputs, returns a map of output name -> stream
func ExecuteMap(ctx *ExecutionContext, nodeID int, nodeName string, config models.MapConfig, inputs ...[]map[string]interface{}) (*RowStream, error) {
	totalInputRows := 0
	for _, input := range inputs {
		totalInputRows += len(input)
	}
	log.Printf("Node %d (%s): Processing %d total input rows", nodeID, nodeName, totalInputRows)

	// Apply join if multiple inputs
	var workingData []map[string]interface{}
	if len(inputs) > 1 && config.Join != nil {
		workingData = applyJoin(config.Join, inputs, config.Inputs)
	} else if len(inputs) > 0 {
		workingData = inputs[0]
	}

	bufferSize := 1000
	stream := NewRowStream(bufferSize)

	go func() {
		defer stream.Close()
		outputCount := 0

		for _, row := range workingData {
			if ctx.IsCancelled() {
				log.Printf("Node %d (%s): Cancelled after %d rows", nodeID, nodeName, outputCount)
				return
			}

			// Build output row based on config
			outRow := make(map[string]interface{})

			if len(config.Outputs) > 0 {
				output := config.Outputs[0]
				for _, col := range output.Columns {
					val, err := computeColumn(col, row, config.Inputs)
					if err != nil {
						stream.SendError(fmt.Errorf("column %s: %w", col.Name, err))
						return
					}
					outRow[col.Name] = val
				}
			} else {
				// Passthrough if no columns defined
				outRow = row
			}

			if !stream.Send(outRow) {
				return
			}
			outputCount++
		}

		log.Printf("Node %d (%s): Transformed %d -> %d rows", nodeID, nodeName, totalInputRows, outputCount)
	}()

	return stream, nil
}

// ExecuteMapMultiOutput executes a Map node with multiple outputs
func ExecuteMapMultiOutput(ctx *ExecutionContext, nodeID int, nodeName string, config models.MapConfig, inputs ...[]map[string]interface{}) (map[string]*RowStream, error) {
	totalInputRows := 0
	for _, input := range inputs {
		totalInputRows += len(input)
	}
	log.Printf("Node %d (%s): Processing %d total input rows to %d outputs", nodeID, nodeName, totalInputRows, len(config.Outputs))

	// Apply join if multiple inputs
	var workingData []map[string]interface{}
	if len(inputs) > 1 && config.Join != nil {
		workingData = applyJoin(config.Join, inputs, config.Inputs)
	} else if len(inputs) > 0 {
		workingData = inputs[0]
	}

	bufferSize := 1000
	outputs := make(map[string]*RowStream)
	for _, output := range config.Outputs {
		outputs[output.Name] = NewRowStream(bufferSize)
	}

	go func() {
		defer func() {
			for _, stream := range outputs {
				stream.Close()
			}
		}()

		for _, row := range workingData {
			if ctx.IsCancelled() {
				return
			}

			// Build output row for each output
			for _, output := range config.Outputs {
				outRow := make(map[string]interface{})
				for _, col := range output.Columns {
					val, err := computeColumn(col, row, config.Inputs)
					if err != nil {
						outputs[output.Name].SendError(fmt.Errorf("column %s: %w", col.Name, err))
						return
					}
					outRow[col.Name] = val
				}
				outputs[output.Name].Send(outRow)
			}
		}

		log.Printf("Node %d (%s): Processed %d rows to %d outputs", nodeID, nodeName, totalInputRows, len(outputs))
	}()

	return outputs, nil
}

// applyJoin applies join logic based on configuration
func applyJoin(join *models.JoinConfig, inputs [][]map[string]interface{}, inputConfigs []models.InputFlow) []map[string]interface{} {
	if len(inputs) < 2 {
		if len(inputs) == 1 {
			return inputs[0]
		}
		return nil
	}

	// Find input indices by name
	leftIdx, rightIdx := 0, 1
	for i, cfg := range inputConfigs {
		if cfg.Name == join.LeftInput {
			leftIdx = i
		}
		if cfg.Name == join.RightInput {
			rightIdx = i
		}
	}

	leftData := inputs[leftIdx]
	rightData := inputs[rightIdx]

	switch join.Type {
	case models.JoinTypeUnion:
		return applyUnion(inputs)
	case models.JoinTypeCross:
		return applyCrossJoin(leftData, rightData)
	case models.JoinTypeInner:
		return applyKeyedJoin(leftData, rightData, join, "inner")
	case models.JoinTypeLeft:
		return applyKeyedJoin(leftData, rightData, join, "left")
	case models.JoinTypeRight:
		return applyKeyedJoin(leftData, rightData, join, "right")
	case models.JoinTypeFull:
		return applyKeyedJoin(leftData, rightData, join, "full")
	default:
		return applyUnion(inputs)
	}
}

func applyUnion(inputs [][]map[string]interface{}) []map[string]interface{} {
	total := 0
	for _, input := range inputs {
		total += len(input)
	}
	result := make([]map[string]interface{}, 0, total)
	for _, input := range inputs {
		result = append(result, input...)
	}
	return result
}

func applyCrossJoin(left, right []map[string]interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(left)*len(right))
	for _, l := range left {
		for _, r := range right {
			merged := make(map[string]interface{})
			for k, v := range l {
				merged[k] = v
			}
			for k, v := range r {
				merged[k] = v
			}
			result = append(result, merged)
		}
	}
	return result
}

func applyKeyedJoin(left, right []map[string]interface{}, join *models.JoinConfig, joinType string) []map[string]interface{} {
	// Build key function
	getKey := func(row map[string]interface{}, keys []string) string {
		if len(keys) == 0 {
			return ""
		}
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = fmt.Sprintf("%v", row[k])
		}
		return strings.Join(parts, "\x00")
	}

	leftKeys := []string{join.LeftKey}
	rightKeys := []string{join.RightKey}
	if len(join.LeftKeys) > 0 {
		leftKeys = join.LeftKeys
		rightKeys = join.RightKeys
	}

	// Build right index
	rightIndex := make(map[string][]map[string]interface{})
	for _, row := range right {
		key := getKey(row, rightKeys)
		rightIndex[key] = append(rightIndex[key], row)
	}

	result := make([]map[string]interface{}, 0)
	matchedRight := make(map[string]bool)

	// Process left side
	for _, l := range left {
		key := getKey(l, leftKeys)
		if rights, ok := rightIndex[key]; ok {
			matchedRight[key] = true
			for _, r := range rights {
				merged := make(map[string]interface{})
				for k, v := range l {
					merged[k] = v
				}
				for k, v := range r {
					merged[k] = v
				}
				result = append(result, merged)
			}
		} else if joinType == "left" || joinType == "full" {
			result = append(result, l)
		}
	}

	// For right/full join, add unmatched right rows
	if joinType == "right" || joinType == "full" {
		for key, rights := range rightIndex {
			if !matchedRight[key] {
				result = append(result, rights...)
			}
		}
	}

	return result
}

// computeColumn computes a single output column value
func computeColumn(col models.MapOutputCol, row map[string]interface{}, inputs []models.InputFlow) (interface{}, error) {
	switch col.FuncType {
	case models.FuncTypeDirect:
		return getColumnValue(col.InputRef, row), nil

	case models.FuncTypeLibrary:
		return executeLibFunc(col.LibFunc, col.Args, row)

	case models.FuncTypeCustom:
		// For custom expressions, we'd need an expression evaluator
		// For now, treat as direct if InputRef is set
		if col.InputRef != "" {
			return getColumnValue(col.InputRef, row), nil
		}
		return nil, fmt.Errorf("custom expressions not yet supported at runtime")

	default:
		if col.InputRef != "" {
			return getColumnValue(col.InputRef, row), nil
		}
		return nil, nil
	}
}

// getColumnValue extracts a column value from "InputName.column" or just "column"
func getColumnValue(ref string, row map[string]interface{}) interface{} {
	parts := strings.SplitN(ref, ".", 2)
	var colName string
	if len(parts) == 2 {
		colName = parts[1]
	} else {
		colName = ref
	}
	return row[colName]
}

// executeLibFunc executes a library function
func executeLibFunc(funcName string, args []models.FuncArg, row map[string]interface{}) (interface{}, error) {
	// Resolve arguments
	resolved := make([]interface{}, len(args))
	for i, arg := range args {
		switch arg.Type {
		case "column":
			resolved[i] = getColumnValue(arg.Value, row)
		case "literal":
			resolved[i] = arg.Value
		default:
			resolved[i] = arg.Value
		}
	}

	// Execute function
	switch funcName {
	// String functions
	case "Concat":
		if len(resolved) > 0 {
			sep := fmt.Sprintf("%v", resolved[0])
			return libConcat(sep, resolved[1:]...), nil
		}
		return "", nil
	case "ConcatNoSep":
		return libConcat("", resolved...), nil
	case "Upper":
		if len(resolved) > 0 {
			return libUpper(resolved[0]), nil
		}
		return "", nil
	case "Lower":
		if len(resolved) > 0 {
			return libLower(resolved[0]), nil
		}
		return "", nil
	case "Trim":
		if len(resolved) > 0 {
			return libTrim(resolved[0]), nil
		}
		return "", nil
	case "ToString":
		if len(resolved) > 0 {
			return libToString(resolved[0]), nil
		}
		return "", nil

	// Numeric functions
	case "Add":
		if len(resolved) >= 2 {
			return libAdd(resolved[0], resolved[1]), nil
		}
		return 0.0, nil
	case "Sub":
		if len(resolved) >= 2 {
			return libSub(resolved[0], resolved[1]), nil
		}
		return 0.0, nil
	case "Mul":
		if len(resolved) >= 2 {
			return libMul(resolved[0], resolved[1]), nil
		}
		return 0.0, nil
	case "Div":
		if len(resolved) >= 2 {
			return libDiv(resolved[0], resolved[1]), nil
		}
		return 0.0, nil
	case "ToInt":
		if len(resolved) > 0 {
			return libToInt(resolved[0]), nil
		}
		return int64(0), nil
	case "ToFloat":
		if len(resolved) > 0 {
			return libToFloat(resolved[0]), nil
		}
		return 0.0, nil

	// Null handling
	case "Coalesce":
		return libCoalesce(resolved...), nil
	case "IfNull":
		if len(resolved) >= 2 {
			return libIfNull(resolved[0], resolved[1]), nil
		}
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown function: %s", funcName)
	}
}

// ============================================================
// LIBRARY FUNCTIONS
// ============================================================

func libConcat(separator string, values ...interface{}) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		if v != nil {
			parts = append(parts, fmt.Sprintf("%v", v))
		}
	}
	return strings.Join(parts, separator)
}

func libUpper(v interface{}) string {
	if v == nil {
		return ""
	}
	return strings.ToUpper(fmt.Sprintf("%v", v))
}

func libLower(v interface{}) string {
	if v == nil {
		return ""
	}
	return strings.ToLower(fmt.Sprintf("%v", v))
}

func libTrim(v interface{}) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func libToString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func libToFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	default:
		f, _ := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
		return f
	}
}

func libAdd(a, b interface{}) float64 {
	return libToFloat64(a) + libToFloat64(b)
}

func libSub(a, b interface{}) float64 {
	return libToFloat64(a) - libToFloat64(b)
}

func libMul(a, b interface{}) float64 {
	return libToFloat64(a) * libToFloat64(b)
}

func libDiv(a, b interface{}) float64 {
	divisor := libToFloat64(b)
	if divisor == 0 {
		return 0
	}
	return libToFloat64(a) / divisor
}

func libRound(v interface{}, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(libToFloat64(v)*multiplier) / multiplier
}

func libAbs(v interface{}) float64 {
	return math.Abs(libToFloat64(v))
}

func libToInt(v interface{}) int64 {
	return int64(libToFloat64(v))
}

func libToFloat(v interface{}) float64 {
	return libToFloat64(v)
}

func libCoalesce(values ...interface{}) interface{} {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func libIfNull(v, defaultVal interface{}) interface{} {
	if v == nil {
		return defaultVal
	}
	return v
}