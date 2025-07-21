package parser

import (
	"errors"
	"strings"
	"testing"
)

func assertNoErrors(t *testing.T, errs []error) {
	t.Helper()
	if len(errs) > 0 {
		t.Fatalf("ParseHTML falló inesperadamente con %d errores: %v", len(errs), errs)
	}
}

func TestParseHTMLSimple(t *testing.T) {
	html := `<html><body><h1>Título</h1><p>Párrafo.</p></body></html>`
	doc, errs := ParseHTML(html)
	assertNoErrors(t, errs)

	if doc.TagName != "html" {
		t.Errorf("Se esperaba la etiqueta raíz 'html', se obtuvo '%s'", doc.TagName)
	}
	if len(doc.Children) != 1 || doc.Children[0].TagName != "body" {
		t.Fatal("Se esperaba que 'html' tuviera un hijo 'body'")
	}
	body := doc.Children[0]
	if len(body.Children) != 2 {
		t.Fatalf("Se esperaba que 'body' tuviera 2 hijos, se obtuvieron %d", len(body.Children))
	}
	h1 := body.Children[0]
	if h1.TagName != "h1" || len(h1.Children) != 1 || h1.Children[0].Type != TextNodeType || h1.Children[0].Data != "Título" {
		t.Error("Fallo al parsear el nodo h1 o su texto")
	}
	p := body.Children[1]
	if p.TagName != "p" || len(p.Children) != 1 || p.Children[0].Type != TextNodeType || p.Children[0].Data != "Párrafo." {
		t.Error("Fallo al parsear el nodo p o su texto")
	}
}

func TestParseHTMLWithAttributes(t *testing.T) {
	html := `<div id="main" class='container' data-id=123><input type="text" disabled/></div>`
	doc, errs := ParseHTML(html)
	assertNoErrors(t, errs)

	if doc.TagName != "div" {
		t.Errorf("Se esperaba 'div', se obtuvo '%s'", doc.TagName)
	}
	if val, ok := doc.Attributes["id"]; !ok || val != "main" {
		t.Errorf("Atributo 'id' incorrecto: se esperaba 'main', se obtuvo '%s'", val)
	}
	if val, ok := doc.Attributes["class"]; !ok || val != "container" {
		t.Errorf("Atributo 'class' incorrecto: se esperaba 'container', se obtuvo '%s'", val)
	}
	if val, ok := doc.Attributes["data-id"]; !ok || val != "123" {
		t.Errorf("Atributo 'data-id' sin comillas incorrecto: se esperaba '123', se obtuvo '%s'", val)
	}

	if len(doc.Children) != 1 {
		t.Fatalf("Se esperaba 1 hijo para el div, se obtuvieron %d", len(doc.Children))
	}
	input := doc.Children[0]
	if input.TagName != "input" {
		t.Errorf("Se esperaba 'input', se obtuvo '%s'", input.TagName)
	}
	if val, ok := input.Attributes["type"]; !ok || val != "text" {
		t.Errorf("Atributo 'type' incorrecto: se esperaba 'text', se obtuvo '%s'", val)
	}
	if _, ok := input.Attributes["disabled"]; !ok {
		t.Error("Se esperaba el atributo booleano 'disabled'")
	}
}

func TestParseHTMLCommentsAndDoctype(t *testing.T) {
	html := `<!DOCTYPE html><!-- Esto es un comentario --><p>Texto.</p>`
	doc, errs := ParseHTML(html)
	assertNoErrors(t, errs)

	if doc.Type != DocumentFragmentNodeType || len(doc.Children) != 3 {
		t.Fatalf("Se esperaba un DocumentFragment con 3 hijos, se obtuvieron %d", len(doc.Children))
	}
	doctype := doc.Children[0]
	if doctype.Type != DoctypeNodeType || doctype.Data != "html" {
		t.Errorf("Fallo al parsear DOCTYPE. Se obtuvo Data='%s'", doctype.Data)
	}
	comment := doc.Children[1]
	if comment.Type != CommentNodeType || comment.Data != " Esto es un comentario " {
		t.Errorf("Fallo al parsear el comentario. Se obtuvo Data='%s'", comment.Data)
	}
	p := doc.Children[2]
	if p.TagName != "p" || len(p.Children) != 1 || p.Children[0].Data != "Texto." {
		t.Error("Fallo al parsear el párrafo después de los comentarios")
	}
}

func TestParseHTMLResilienceUnclosedTag(t *testing.T) {
	html := `<div><p>Texto sin cerrar`
	doc, errs := ParseHTML(html)

	if len(errs) != 2 {
		t.Fatalf("Se esperaban 2 errores de cierre implícito (p y div), se obtuvieron %d: %v", len(errs), errs)
	}

	expectedError1 := "etiqueta '<p>' no fue cerrada"
	if !strings.Contains(errs[0].Error(), expectedError1) {
		t.Errorf("El primer error no fue el esperado. Se obtuvo: '%s'", errs[0])
	}

	expectedError2 := "etiqueta '<div>' no fue cerrada"
	if !strings.Contains(errs[1].Error(), expectedError2) {
		t.Errorf("El segundo error no fue el esperado. Se obtuvo: '%s'", errs[1])
	}

	if doc.TagName != "div" || len(doc.Children) != 1 {
		t.Fatal("Se esperaba un div con 1 hijo")
	}
	p := doc.Children[0]
	if p.TagName != "p" || len(p.Children) != 1 || p.Children[0].Data != "Texto sin cerrar" {
		t.Error("La etiqueta 'p' no se parseó correctamente dentro del 'div'")
	}
}

func TestParseHTMLResilienceWrongClosingTag(t *testing.T) {
	html := `<div><p>Texto</div>`
	doc, errs := ParseHTML(html)

	if len(errs) != 1 {
		t.Fatalf("Se esperaba 1 error de cierre implícito, se obtuvieron %d: %v", len(errs), errs)
	}

	expectedErr1 := "etiqueta de cierre implícita para '<p>' debido a la etiqueta de cierre explícita de '</div>'"
	if !strings.Contains(errs[0].Error(), expectedErr1) {
		t.Errorf("El primer error no fue el esperado. Se obtuvo: '%s'", errs[0].Error())
	}
	if doc == nil {
		t.Fatal("No se debería haber devuelto un árbol nulo para este error recuperable.")
	}
	if doc.TagName != "div" {
		t.Fatal("Se esperaba un div")
	}
	if len(doc.Children) != 1 || doc.Children[0].TagName != "p" {
		t.Fatal("Se esperaba que 'p' fuera hijo de 'div'")
	}
}

func TestMultipleRootElements(t *testing.T) {
	html := `<p>Uno</p><p>Dos</p>`
	doc, errs := ParseHTML(html)
	assertNoErrors(t, errs)

	if doc.Type != DocumentFragmentNodeType {
		t.Errorf("Se esperaba un DocumentFragment, se obtuvo %v", doc.Type)
	}
	if len(doc.Children) != 2 {
		t.Fatalf("Se esperaban 2 elementos raíz, se obtuvieron %d", len(doc.Children))
	}
	p1, p2 := doc.Children[0], doc.Children[1]
	if p1.TagName != "p" || len(p1.Children) != 1 || p1.Children[0].Data != "Uno" {
		t.Error("Fallo al parsear el primer párrafo")
	}
	if p2.TagName != "p" || len(p2.Children) != 1 || p2.Children[0].Data != "Dos" {
		t.Error("Fallo al parsear el segundo párrafo")
	}
}

// --- Tests de Casos de Error ---

func TestParseHTMLEmptyInput(t *testing.T) {
	html := "   "
	doc, errs := ParseHTML(html)
	if len(errs) != 1 || !errors.Is(errs[0], ErrEmptyInput) {
		t.Errorf("Se esperaba un slice con ErrEmptyInput, se obtuvo: %v", errs)
	}
	if doc != nil {
		t.Error("El nodo devuelto debería ser nil en caso de error fatal")
	}
}

func TestParseHTMLUnterminatedTag(t *testing.T) {
	html := `<div id="main"`
	doc, errs := ParseHTML(html)
	if len(errs) != 1 || !errors.Is(errs[0], ErrUnterminatedTag) {
		t.Errorf("Se esperaba un slice con ErrUnterminatedTag, se obtuvo: %v", errs)
	}
	if doc != nil {
		t.Error("El nodo devuelto debería ser nil en caso de error fatal")
	}
}

func TestParseHTMLUnterminatedComment(t *testing.T) {
	html := `<!-- un comentario sin fin`
	doc, errs := ParseHTML(html)
	if len(errs) != 1 || !errors.Is(errs[0], ErrUnterminatedComment) {
		t.Errorf("Se esperaba un slice con ErrUnterminatedComment, se obtuvo: %v", errs)
	}
	if doc != nil {
		t.Error("El nodo devuelto debería ser nil en caso de error fatal")
	}
}

func TestParseHTMLResilienceMalformedAttribute(t *testing.T) {
	html := `<div id="main" / class="container"></div>`
	doc, errs := ParseHTML(html)

	if len(errs) != 1 {
		t.Fatalf("Se esperaba 1 error no fatal, se obtuvieron %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "carácter inválido '/'") {
		t.Errorf("El mensaje de error no fue el esperado: %v", errs[0])
	}

	if doc == nil {
		t.Fatal("El parser no debería devolver un árbol nulo para errores recuperables.")
	}
	if doc.TagName != "div" {
		t.Errorf("Se esperaba 'div', se obtuvo '%s'", doc.TagName)
	}
	if val, ok := doc.Attributes["id"]; !ok || val != "main" {
		t.Errorf("Atributo 'id' incorrecto: se esperaba 'main', se obtuvo '%s'", val)
	}
	if val, ok := doc.Attributes["class"]; !ok || val != "container" {
		t.Errorf("Atributo 'class' incorrecto: se esperaba 'container', se obtuvo '%s'", val)
	}
	if len(doc.Attributes) != 2 {
		t.Errorf("Se esperaban 2 atributos, se obtuvieron %d", len(doc.Attributes))
	}
}

func TestParseSelfClosingTag(t *testing.T) {
	html := `<div><br/><p>Texto después de auto-cierre</p></div>`
	doc, errs := ParseHTML(html)
	assertNoErrors(t, errs)

	if doc.TagName != "div" {
		t.Fatalf("Se esperaba la etiqueta raíz 'div', se obtuvo '%s'", doc.TagName)
	}

	if len(doc.Children) != 2 {
		t.Fatalf("Se esperaba que 'div' tuviera 2 hijos (<br> y <p>), se obtuvieron %d", len(doc.Children))
	}

	br := doc.Children[0]
	if br.TagName != "br" {
		t.Errorf("Se esperaba que el primer hijo fuera 'br', se obtuvo '%s'", br.TagName)
	}

	if len(br.Children) != 0 {
		t.Errorf("Se esperaba que la etiqueta 'br' no tuviera hijos, se obtuvieron %d", len(br.Children))
	}

	p := doc.Children[1]
	if p.TagName != "p" {
		t.Errorf("Se esperaba que el segundo hijo fuera 'p', se obtuvo '%s'", p.TagName)
	}
	if len(p.Children) != 1 || p.Children[0].Type != TextNodeType || p.Children[0].Data != "Texto después de auto-cierre" {
		t.Error("El párrafo después de la etiqueta de auto-cierre no se parseó correctamente.")
	}
}
