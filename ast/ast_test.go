package ast

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Konstantin8105/c4go/util"
	"github.com/stretchr/testify/assert"
)

func formatMultiLine(o interface{}) string {
	s := fmt.Sprintf("%#v", o)
	s = strings.Replace(s, "{", "{\n", -1)
	s = strings.Replace(s, ", ", "\n", -1)
	return s
}

func runNodeTests(t *testing.T, tests map[string]Node) {
	i := 1
	for line, expected := range tests {
		testName := fmt.Sprintf("Example%d", i)
		i++

		t.Run(testName, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Panic for: %v, %v\n%v", testName, line, r)
				}
			}()

			name := reflect.TypeOf(expected).Elem().Name()
			actual := Parse(name+" "+line, 0)

			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("%s", util.ShowDiff(formatMultiLine(expected), formatMultiLine(actual)))
			}
			if actual == nil {
				t.Errorf("Expected non-nil node for %s", name)
				return
			}
			if int64(actual.Address()) == 0 {
				t.Errorf("Address for test cannot be zero: %v", actual.Address())
			}
			if len(actual.Children()) != 0 {
				t.Errorf("Amount of children cannot be more than 0 initially")
			}
			actual.AddChild(nil)
			if len(actual.Children()) != 1 {
				t.Errorf("Amount of children must be 1 after adding")
			}
			if actual.Children()[0] != nil {
				t.Errorf("Child must be nil")
			}
			pos := actual.Position()
			if pos.Line == 0 && pos.Column == 0 && pos.LineEnd == 0 && pos.ColumnEnd == 0 && pos.File == "" {
				t.Log("Consider testing with non-zero position")
			}
			if pos.Line < 0 || pos.Column < 0 || pos.LineEnd < 0 || pos.ColumnEnd < 0 {
				t.Errorf("Negative position not acceptable")
			}
			var posC Position
			posC.Line = -1
			setPosition(actual, posC)
			if actual.Position().Line != -1 {
				t.Log("Cannot change position")
			}
		})
	}
}

func TestPrint(t *testing.T) {
	cond := &ConditionalOperator{addr: 0x12345678}
	cond.AddChild(&ImplicitCastExpr{addr: 0x12345679})
	cond.AddChild(&ImplicitCastExpr{addr: 0x1234567a})
	s := Atos(cond)
	if len(s) == 0 {
		t.Fatalf("Cannot convert AST tree: %#v", cond)
	}
	lines := strings.Split(s, "\n")
	var amount int
	for _, l := range lines {
		if strings.Contains(l, "ImplicitCastExpr") {
			amount++
		}
	}
	if amount != 2 {
		t.Errorf("Expected 2 ImplicitCastExpr nodes, got %d", amount)
	}
}

func TestPanicCheck(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for invalid string line")
		}
	}()
	Parse("Some strange line", 0)
}

func TestNullStmt(t *testing.T) {
	n := Parse("NullStmt", 0)
	assert.Nil(t, n, "Expected nil for NullStmt")
}

func TestAstNodes(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "ast.go", nil, parser.DeclarationErrors)
	if err != nil {
		t.Fatalf("%v", err)
	}

	var fr Founder
	ast.Walk(fr, f.Decls[4])

	nodesFromAst = append(nodesFromAst, "")

	for _, c := range nodesFromAst {
		t.Run(fmt.Sprintf("%v", c), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Cannot parse: %v", r)
				}
			}()
			Parse(c, 0)
		})
	}

	dat, err := ioutil.ReadFile("position.go")
	if err != nil {
		t.Fatalf("Error reading `position.go`: %v", err)
	}

	index := bytes.Index(dat, []byte("setPosition"))
	if index < 0 {
		t.Fatalf("Cannot find function `setPosition`")
	}
	dat = dat[index:]

	for _, c := range nodesFromAst {
		if c == "" || c == "\"NullStmt\"" {
			continue
		}
		t.Run(fmt.Sprintf("%v", c), func(t *testing.T) {
			c, err := strconv.Unquote(c)
			if err != nil {
				t.Fatalf("Unquote invalid: %v", err)
			}
			index := bytes.Index(dat, []byte(c))
			if index < 0 {
				t.Fatalf("Cannot find type: %v", c)
			}
		})
	}
}

// Additional tests for #506
func TestUnknownNodeTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
	}{
		{
			name:     "TypeVisibilityAttr",
			input:    "TypeVisibilityAttr 0x2962acf3b20 <<invalid sloc>> Implicit Default",
			wantType: "TypeVisibilityAttr",
		},
		{
			name:     "MSAllocatorAttr",
			input:    "MSAllocatorAttr 0x2962aee6218 <line:190:25>",
			wantType: "MSAllocatorAttr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Parse(tt.input, 0)
			assert.NotNil(t, node, "Expected non-nil node")
			unknown, ok := node.(*UnknownNode)
			assert.True(t, ok, "Expected UnknownNode")
			assert.Equal(t, tt.wantType, unknown.TypeName, "Expected correct type name")
			assert.Equal(t, ParseAddress(strings.Split(tt.input, " ")[1]), unknown.Address(), "Expected correct address")
		})
	}
}
func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  string
		wantAddr  Address
		wantLine  int
		wantError bool
	}{
		{
			name:      "Empty input",
			input:     "",
			wantType:  "",
			wantError: false,
		},
		{
			name:      "TranslationUnitDecl",
			input:     "TranslationUnitDecl 0x12345678 <line:1:1>",
			wantType:  "TranslationUnitDecl",
			wantAddr:  0x12345678,
			wantLine:  1,
			wantError: false,
		},
		{
			name:      "FunctionDecl",
			input:     "FunctionDecl 0x98765432 <line:10:5>",
			wantType:  "FunctionDecl",
			wantAddr:  0x98765432,
			wantLine:  10,
			wantError: false,
		},
		{
			name:      "TypeVisibilityAttr",
			input:     "TypeVisibilityAttr 0x2962acf3b20 <<invalid sloc>> Implicit Default",
			wantType:  "UnknownNode",
			wantAddr:  0x2962acf3b20,
			wantLine:  -1,
			wantError: false,
		},
		{
			name:      "MSAllocatorAttr",
			input:     "MSAllocatorAttr 0x2962aee6218 <line:190:25>",
			wantType:  "UnknownNode",
			wantAddr:  0x2962aee6218,
			wantLine:  190,
			wantError: false,
		},
		{
			name:      "Invalid type",
			input:     "InvalidType 0x11111111 <line:5:5>",
			wantType:  "",
			wantError: true,
		},
		{
			name:      "NullStmt",
			input:     "NullStmt",
			wantType:  "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := Parse(tt.input, 0)
			if tt.wantError {
				assert.Error(t, err, "Expected an error")
				assert.Nil(t, node, "Expected nil node on error")
				return
			}
			assert.NoError(t, err, "Unexpected error")

			if tt.wantType == "" {
				assert.Nil(t, node, "Expected nil node")
				return
			}

			assert.NotNil(t, node, "Expected non-nil node")
			switch n := node.(type) {
			case *UnknownNode:
				assert.Equal(t, tt.wantType, "UnknownNode", "Expected UnknownNode")
				assert.Equal(t, tt.input, n.Raw, "Expected raw input match")
			case *TranslationUnitDecl:
				assert.Equal(t, tt.wantType, "TranslationUnitDecl", "Expected TranslationUnitDecl")
			case *FunctionDecl:
				assert.Equal(t, tt.wantType, "FunctionDecl", "Expected FunctionDecl")
			default:
				t.Errorf("Unexpected node type: %T", node)
			}

			assert.Equal(t, tt.wantAddr, node.Address(), "Address mismatch")
			assert.Equal(t, tt.wantLine, node.Position().Line, "Line mismatch")
		})
	}
}
