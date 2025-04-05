package main

import (
    "bytes"
    "fmt"
    "go/ast"
    "go/format"
    "go/parser"
    "go/printer"
    "go/token"
    "os"
    "path/filepath"
    "strings" // Added missing import
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: go run fixlonglines.go <directory>")
        os.Exit(1)
    }

    dir := os.Args[1]
    err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        // Skip directories and non-.go files
        if info.IsDir() || filepath.Ext(path) != ".go" {
            return nil
        }

        if err := processFile(path); err != nil {
            fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", path, err)
            return nil // Continue with other files despite errors
        }
        fmt.Printf("Formatted %s\n", path)
        return nil
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error walking directory %s: %v\n", dir, err)
        os.Exit(1)
    }
}

func processFile(filename string) error {
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
    if err != nil {
        return fmt.Errorf("parsing file: %v", err)
    }

    // Transform long lines
    changed := false
    ast.Inspect(file, func(n ast.Node) bool {
        if call, ok := n.(*ast.CallExpr); ok {
            start := fset.Position(call.Pos())
            end := fset.Position(call.End())
            line := getLine(fset, filename, start, end)
            if len(line) > 100 {
                // Mark for multi-line formatting
                call.Lparen = call.Fun.End() + 1 // Move opening paren to next line
                for i := range call.Args {
                    if i > 0 {
                        // Simulate new line by adjusting positions
                        switch arg := call.Args[i].(type) {
                        case *ast.BasicLit:
                            arg.ValuePos = call.Args[i-1].End() + 1
                        case *ast.CallExpr:
                            arg.Lparen = call.Args[i-1].End() + 1
                        }
                    }
                }
                call.Rparen = call.Args[len(call.Args)-1].End() + 1
                changed = true
            }
        }
        return true
    })

    if !changed {
        return nil // No changes needed
    }

    // Write formatted output
    var buf bytes.Buffer
    config := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 4}
    if err := config.Fprint(&buf, fset, file); err != nil {
        return fmt.Errorf("formatting: %v", err)
    }
    formatted, err := format.Source(buf.Bytes())
    if err != nil {
        return fmt.Errorf("applying gofmt: %v", err)
    }
    if err := os.WriteFile(filename, formatted, 0644); err != nil {
        return fmt.Errorf("writing file: %v", err)
    }
    return nil
}

func getLine(fset *token.FileSet, filename string, start, end token.Position) string {
    data, err := os.ReadFile(filename)
    if err != nil {
        return ""
    }
    lines := strings.Split(string(data), "\n")
    if start.Line-1 >= len(lines) || start.Line < 1 {
        return "" // Invalid line number
    }
    line := lines[start.Line-1]

    // Safely handle column bounds
    startCol := start.Column - 1
    endCol := end.Column
    if startCol < 0 {
        startCol = 0
    }
    if endCol > len(line) {
        endCol = len(line)
    }
    if startCol >= endCol || startCol >= len(line) {
        return strings.TrimSpace(line) // Return full line if bounds are invalid
    }

    return strings.TrimSpace(line[startCol:endCol])
}