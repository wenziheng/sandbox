package stevenson

import(
  "fmt"
  "os"
  "unicode/utf8"
  "strings"
  "io/ioutil"
)

type LiquidDocument struct {
  Source string
  Document LiquidNode
  Context map[interface{}]interface{}
}

// nodes can be:
//  literal text  <ul id="products">
//  expression:   if/for
//  markup:       {{ page.title }}
type LiquidNode struct {
  TextContent string
  Children    []LiquidNode
}

type ItemType int

const (
  itemError ItemType = iota // error occurred
  itemText
  itemMarkupStart
  itemMarkup
  itemMarkupEnd
  itemExpressionStart
  itemExpression
  itemExpressionEnd
  itemInclude
  itemEOF
)

type Item struct {
  typ ItemType
  val string
}

type Lexer struct {
  Input string
  Start int
  Pos int
  Width int
  Items chan Item
}

type stateFn func(*Lexer) stateFn
var eof = -1

func (self Item) String () string {
  switch self.typ {
  // case itemText:
  //   return Item.val
  case itemMarkupStart:
    return "{{"
  case itemMarkup:
    return fmt.Sprintf("[[Markup:%s]]", self.val)
  case itemMarkupEnd:
    return "}}"
  case itemExpressionStart:
    return "{%"
  case itemExpression:
    return fmt.Sprintf("[[Expression:%s]]", self.val)
  case itemExpressionEnd:
    return "%}"
  case itemEOF:
    return "EOF"
  }

  return fmt.Sprintf("[[Text:%s]]",self.val)
}

func (l *Lexer) next() (r rune, eof bool) {
  if l.Pos >= len(l.Input) {
    l.Width = 0
    return r, true
  }

  r, l.Width = utf8.DecodeRuneInString(l.Input[l.Pos:])
  l.Pos += l.Width
  return r, false
}

func LexExpressionEnd(l *Lexer) stateFn {
  l.Pos += len("}}")
  l.emit(itemExpressionEnd)
  return LexText
}

func LexExpression(l *Lexer) stateFn {
  for {
    if strings.HasPrefix(l.Input[l.Pos:], "}}") {
      if l.Pos > l.Start {
        l.emit(itemExpression)
        return LexExpressionEnd
      }
    }

    if _, eof := l.next(); eof {
      fmt.Printf("Error: unclosed expression!\n")
      l.emit(itemError)
      return nil
    }
  }

  fmt.Printf("Error: unclosed expression!\n")
  l.emit(itemError)
  return nil
}

func LexExpressionStart(l *Lexer) stateFn {
  l.Pos += len("{{")
  l.emit(itemExpressionStart)
  return LexExpression
}

func LexMarkupEnd(l *Lexer) stateFn {
  l.Pos += len("%}")
  l.emit(itemMarkupEnd)
  return LexText
}

func LexMarkup(l *Lexer) stateFn {
  for {
    if strings.HasPrefix(l.Input[l.Pos:], "%}") {
      if l.Pos > l.Start {
        l.emit(itemMarkup)
        return LexMarkupEnd
      }
    }

    /*
    if strings.HasPrefix(l.Input[l.Pos:], "include") {
      if l.Pos > l.Start {
        l.emit(itemInclude)
        return LexMarkupEnd
      }
    }
    */

    if _, eof := l.next(); eof {
      fmt.Printf("Error: unclosed markup!\n")
      l.emit(itemError)
      return nil
    }
  }

  fmt.Printf("Error: unclosed markup!\n")
  l.emit(itemError)
  return nil
}

func LexMarkupStart(l *Lexer) stateFn {
  l.Pos += len("}}")
  l.emit(itemMarkupStart)
  return LexMarkup
}

func LexText(l *Lexer) stateFn {
  for {
    if strings.HasPrefix(l.Input[l.Pos:], "{{") {
      if l.Pos > l.Start {
        l.emit(itemText)
        return LexExpressionStart
      }
    }

    if strings.HasPrefix(l.Input[l.Pos:], "{%") {
      if l.Pos > l.Start {
        l.emit(itemText)
        return LexMarkupStart
      }
    }

    if _, eof := l.next(); eof {
      break
    }
  }

  // correctly reached EOF
  if l.Pos > l.Start {
    l.emit(itemText)
  }

  l.emit(itemEOF)

  return nil
}

func (l *Lexer) emit(t ItemType) {
  l.Items <- Item{t, l.Input[l.Start:l.Pos]}
  l.Start = l.Pos
}

func LexDocument(tmpl string) (lexer *Lexer) {
  lexer = &Lexer{
    Input: tmpl,
    Pos:  0,
    Items: make(chan Item),
  }

  go func (l *Lexer) {
    for state := LexText; state != nil; {
      state = state(l)
    }
    fmt.Printf("LexDocument: reached EOF, closing channel\n")
    close(l.Items)
  }(lexer)


  // consume all the Items
  for item := range(lexer.Items) {
    fmt.Printf("Lexed: %q\n", item)
    if item.typ == itemEOF {
      break
    }
    if item.typ == itemError {
      fmt.Printf("Lexing failed!\n")
      break
    }
  }

  return
}

func NewLiquidDocument(tmpl string) LiquidDocument {
  lexer := LexDocument(tmpl)
  fmt.Printf("Lexer: %q\n", lexer)
  return LiquidDocument{
    Source:   tmpl,
    Document: LiquidNode{
      TextContent: tmpl,
    },
  }
}

func (self LiquidDocument) Render(context map[interface{}]interface{}) string {
  return self.Source
}

// TODO: pass along and merge contexts
func ProcessLiquid(fi FileInfo) (string, error) {
  content, err := fi.GetContent()
  if err != nil {
    return "", err
  }

  doc := NewLiquidDocument(content)
  context := make(map[interface{}]interface{})
  return doc.Render(context), nil
}

func ProcessLiquidFileWithFrontMatter(fi FileInfo) error {
  fmt.Printf("ProcessLiquidFileWithFrontMatter:\n")

  result, err := ProcessLiquid(fi)
  if err != nil {
    return err
  }

  statInfo, err := os.Stat(fi.SrcPath)
  if err != nil {
    return err
  }

  ioutil.WriteFile(fi.DstPath, []byte(result), statInfo.Mode())
  return nil
}

func ProcessLiquidFile(fi FileInfo) error {
  fmt.Printf("ProcessLiquidFile:\n")

  result, err := ProcessLiquid(fi)
  if err != nil {
    return err
  }
  statInfo, err := os.Stat(fi.SrcPath)
  if err != nil {
    return err
  }

  ioutil.WriteFile(fi.DstPath, []byte(result), statInfo.Mode())
  return nil
}

