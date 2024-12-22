package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var (
	ErrInvalidExpression   = errors.New("invalid expression")
	ErrDivisionByZero      = errors.New("division by zero")
	ErrExpressionSyntax    = errors.New("expression is not correct")
	ErrNotPostMethod       = errors.New("method not allowed")
	ErrInternalServerError = errors.New("internal server error")
)

// структура для группировки данных
type RequestBody struct {
	Expression string `json:"expression"`
}

// структура для ответа, если все без ошибок
type SuccessResponse struct {
	Result string `json:"result"`
}

// структура для ошибок
type ErrorResponse struct {
	Error string `json:"error"`
}

// Операции и их приоритеты
var operations = map[string]int{
	"+": 1,
	"-": 1,
	"*": 2,
	"/": 2,
}

// Функция, которая принимает на вход строку и проверяет, является ли эта строка оператором
func isOperator(s string) bool {
	_, ok := operations[s]
	return ok
}

// Функция, которая принимает оператор в виде строки и возвращает его приоритет
func precedence(op string) int {
	prio, _ := operations[op]
	return prio
}

// ConvertToPostfix преобразует выражение из инфиксной формы в постфиксную
func convertToPostfix(infix []string) ([]string, error) {
	var output []string
	stack := make([]string, 0)
	for _, token := range infix {
		if token == "(" {
			stack = append(stack, token)
		} else if token == ")" {
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 || stack[len(stack)-1] != "(" {
				return nil, ErrExpressionSyntax
			}
			stack = stack[:len(stack)-1] // удалить '('
		} else if isOperator(token) {
			for len(stack) > 0 && stack[len(stack)-1] != "(" && precedence(token) <= precedence(stack[len(stack)-1]) {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, token)
		} else { // число
			output = append(output, token)
		}
	}
	for len(stack) > 0 {
		if stack[len(stack)-1] == "(" {
			return nil, ErrExpressionSyntax
		}
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}
	return output, nil
}

// Calculate выполняет вычисления над постфиксным выражением
func calculate(postfix []string) (float64, error) {
	stack := make([]float64, 0)
	for _, token := range postfix {
		if isOperator(token) {
			if len(stack) < 2 {
				return 0, ErrExpressionSyntax
			}
			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2]
			switch token {
			case "+":
				stack = append(stack, a+b)
			case "-":
				stack = append(stack, a-b)
			case "*":
				stack = append(stack, a*b)
			case "/":
				if b == 0 {
					return 0, ErrDivisionByZero
				}
				stack = append(stack, a/b)
			default:
				return 0, ErrExpressionSyntax
			}
		} else {
			value, err := strconv.ParseFloat(token, 64)
			if err != nil {
				return 0, ErrExpressionSyntax
			}
			stack = append(stack, value)
		}
	}
	if len(stack) != 1 {
		return 0, ErrInvalidExpression
	}
	return stack[0], nil
}

// Calc вычисляет значение выражения
func Calc(expression string) (float64, error) {
	if len(expression) < 3 {
		return 0, ErrExpressionSyntax
	}
	tokens := strings.Split(strings.ReplaceAll(expression, " ", ""), "")
	postfix, err := convertToPostfix(tokens)
	if err != nil {
		return 0, err
	}
	result, err := calculate(postfix)
	if err != nil {
		return 0, err
	}
	return result, nil
}

// Структура для чтения выражения из запроса
type Expression struct {
	Data string `json:"expression"`
}

// Структура для ответа без ошибок
type GoodAnswer struct {
	Result float64 `json:"result"`
}

// Структура для ответа с ошибкой
type BadAnswer struct {
	Error string `json:"error"`
}

// Handler для обработки запроса
func CalculateHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем что это Post запрос
	if r.Method != http.MethodPost {
		http.Error(w, ErrNotPostMethod.Error(), http.StatusMethodNotAllowed)
		return
	}
	// Считыем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, ErrInternalServerError.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Распарсим тело запроса в структуру
	var expressionBody Expression
	if err := json.Unmarshal(body, &expressionBody); err != nil {
		errorBytes, err := json.Marshal(&BadAnswer{
			Error: "Invalid request body",
		})
		if err != nil {
			http.Error(w, ErrExpressionSyntax.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, string(errorBytes), http.StatusBadRequest)
		return
	}

	result, errr := Calc(expressionBody.Data)

	if errr != nil {
		errorBytes, err := json.Marshal(&BadAnswer{
			Error: errr.Error(),
		})
		if err != nil {
			http.Error(w, ErrExpressionSyntax.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, string(errorBytes), http.StatusUnprocessableEntity)
	} else {
		resultBytes, err := json.Marshal(&GoodAnswer{
			Result: result,
		})
		if err != nil {
			http.Error(w, ErrExpressionSyntax.Error(), http.StatusOK)
			return
		}
		fmt.Fprint(w, string(resultBytes))
	}
}

func main() {
	mux := http.NewServeMux()
	calculateMux := http.HandlerFunc(CalculateHandler)
	mux.Handle("/api/v1/calculate", calculateMux)
	if err := http.ListenAndServe(":8000", mux); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
