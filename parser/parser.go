package parser

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var (
	ErrEmptyInput          = errors.New("la entrada HTML no puede estar vacía")
	ErrUnterminatedTag     = errors.New("etiqueta no terminada: se alcanzó el final del archivo sin encontrar '>'")
	ErrUnterminatedComment = errors.New("comentario no terminado: se alcanzó el final del archivo sin encontrar '-->'")
)

type NodeType int

const (
	ErrorNodeType NodeType = iota
	TextNodeType
	ElementNodeType
	CommentNodeType
	DoctypeNodeType
	DocumentFragmentNodeType
)

type Node struct {
	Type       NodeType
	TagName    string
	Attributes map[string]string
	Data       string
	Parent     *Node
	Children   []*Node
}

type parser struct {
	pos    int
	input  string
	stack  []*Node
	errors []error
}

var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true,
	"img": true, "input": true, "link": true, "meta": true, "param": true, "source": true,
	"track": true, "wbr": true,
}

func ParseHTML(input string) (*Node, []error) {
	if strings.TrimSpace(input) == "" {
		return nil, []error{ErrEmptyInput}
	}

	root := &Node{Type: DocumentFragmentNodeType, Children: []*Node{}}
	p := &parser{
		input:  input,
		stack:  []*Node{root},
		errors: make([]error, 0),
	}

	p.parseNodes()

	if len(p.stack) > 1 {
		for i := len(p.stack) - 1; i > 0; i-- {
			unclosedTag := p.stack[i].TagName
			p.errors = append(p.errors, fmt.Errorf("etiqueta '<%s>' no fue cerrada; cierre generado implícitamente al final del documento", unclosedTag))
		}
	}

	var finalNode *Node
	var finalErrors []error = p.errors

	isFatal := false
	if len(p.errors) > 0 {
		for _, err := range p.errors {
			if errors.Is(err, ErrUnterminatedTag) || errors.Is(err, ErrUnterminatedComment) {
				isFatal = true
				break
			}
		}
	}

	if !isFatal {
		if len(root.Children) == 1 {
			finalNode = root.Children[0]
		} else {
			finalNode = root
		}
	}

	return finalNode, finalErrors
}

func (p *parser) parseNodes() {
	for p.pos < len(p.input) {
		p.consumeWhitespace()
		if p.pos >= len(p.input) {
			break
		}

		if len(p.stack) == 0 {
			p.errors = append(p.errors, errors.New("estructura de anidamiento inválida, no hay nodo padre en la pila"))
			return
		}
		parent := p.stack[len(p.stack)-1]

		if strings.HasPrefix(p.input[p.pos:], "<") {
			var err error
			if strings.HasPrefix(p.input[p.pos:], "</") {
				p.parseCloseTag()
			} else if strings.HasPrefix(p.input[p.pos:], "<!--") {
				err = p.parseComment(parent)
			} else if strings.HasPrefix(p.input[p.pos:], "<!DOCTYPE") {
				p.parseDoctype(parent)
			} else {
				err = p.parseElement(parent)
			}
			if err != nil {
				p.errors = append(p.errors, err)
				return
			}
		} else {
			p.parseText(parent)
		}
	}
}

func (p *parser) parseElement(parent *Node) error {
	p.pos++
	tagName := p.parseTagName()
	if tagName == "" {
		p.errors = append(p.errors, fmt.Errorf("etiqueta inválida sin nombre en la posición %d", p.pos))
		return nil
	}

	attributes, isSelfClosing, err := p.parseAttributes()
	if err != nil {
		return err
	}

	node := &Node{
		Type:       ElementNodeType,
		TagName:    tagName,
		Attributes: attributes,
		Parent:     parent,
		Children:   []*Node{},
	}
	parent.Children = append(parent.Children, node)

	if !isSelfClosing && !voidElements[tagName] {
		p.stack = append(p.stack, node)
	}

	return nil
}

func (p *parser) parseTagName() string {
	p.consumeWhitespace()
	start := p.pos
	for p.pos < len(p.input) && !unicode.IsSpace(rune(p.input[p.pos])) && !strings.ContainsRune(">/", rune(p.input[p.pos])) {
		p.pos++
	}
	return p.input[start:p.pos]
}

func (p *parser) parseAttributeName() string {
	p.consumeWhitespace()
	start := p.pos
	for p.pos < len(p.input) && !unicode.IsSpace(rune(p.input[p.pos])) && !strings.ContainsRune("=>/", rune(p.input[p.pos])) {
		p.pos++
	}
	return p.input[start:p.pos]
}

func (p *parser) parseAttributes() (map[string]string, bool, error) {
	attributes := make(map[string]string)
	for {
		p.consumeWhitespace()

		if p.pos >= len(p.input) {
			return nil, false, ErrUnterminatedTag
		}

		if p.input[p.pos] == '>' {
			p.pos++
			return attributes, false, nil
		}
		if strings.HasPrefix(p.input[p.pos:], "/>") {
			p.pos += 2
			return attributes, true, nil
		}

		key := p.parseAttributeName()
		if key == "" {
			errMsg := fmt.Sprintf("carácter inválido '%c' encontrado al parsear atributos en la posición %d", p.input[p.pos], p.pos)
			p.errors = append(p.errors, errors.New(errMsg))
			p.pos++
			continue
		}

		p.consumeWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] != '=' {
			attributes[key] = ""
			continue
		}

		p.pos++
		p.consumeWhitespace()
		if p.pos >= len(p.input) {
			return nil, false, ErrUnterminatedTag
		}

		var value string
		quote := p.input[p.pos]

		if quote == '"' || quote == '\'' {
			p.pos++
			start := p.pos
			end := strings.IndexRune(p.input[p.pos:], rune(quote))
			if end == -1 {
				return nil, false, ErrUnterminatedTag
			}
			value = p.input[start : p.pos+end]
			p.pos += end + 1
		} else {
			start := p.pos
			for p.pos < len(p.input) && !unicode.IsSpace(rune(p.input[p.pos])) && p.input[p.pos] != '>' {
				p.pos++
			}
			value = p.input[start:p.pos]
		}
		attributes[key] = value
	}
}

func (p *parser) parseCloseTag() {
	p.pos += 2
	tagName := p.parseTagName()
	p.consumeWhitespace()
	if p.pos < len(p.input) && p.input[p.pos] == '>' {
		p.pos++
	}

	if tagName == "" {
		return
	}

	foundPos := -1
	for i := len(p.stack) - 1; i >= 0; i-- {
		if p.stack[i].TagName == tagName {
			foundPos = i
			break
		}
	}

	if foundPos != -1 {
		for i := len(p.stack) - 1; i > foundPos; i-- {
			unclosedTag := p.stack[i].TagName
			p.errors = append(p.errors, fmt.Errorf("etiqueta de cierre implícita para '<%s>' debido a la etiqueta de cierre explícita de '</%s>'", unclosedTag, tagName))
		}
		p.stack = p.stack[:foundPos]
	} else {
		p.errors = append(p.errors, fmt.Errorf("etiqueta de cierre inesperada '</%s>' sin etiqueta de apertura coincidente", tagName))
	}
}

func (p *parser) parseText(parent *Node) {
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != '<' {
		p.pos++
	}
	text := p.input[start:p.pos]

	if strings.TrimSpace(text) != "" {
		node := &Node{
			Type:   TextNodeType,
			Data:   text,
			Parent: parent,
		}
		parent.Children = append(parent.Children, node)
	}
}

func (p *parser) parseComment(parent *Node) error {
	p.pos += 4
	start := p.pos
	end := strings.Index(p.input[p.pos:], "-->")
	if end == -1 {
		p.pos = len(p.input)
		return ErrUnterminatedComment
	}
	p.pos += end + 3

	node := &Node{
		Type:   CommentNodeType,
		Data:   p.input[start : start+end],
		Parent: parent,
	}
	parent.Children = append(parent.Children, node)
	return nil
}

func (p *parser) parseDoctype(parent *Node) {
	p.pos += len("<!DOCTYPE")
	p.consumeWhitespace()
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != '>' {
		p.pos++
	}

	node := &Node{
		Type:   DoctypeNodeType,
		Data:   strings.TrimSpace(p.input[start:p.pos]),
		Parent: parent,
	}
	parent.Children = append(parent.Children, node)

	if p.pos < len(p.input) {
		p.pos++
	}
}

func (p *parser) consumeWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}
