package main

import (
	"fmt"
	"html-parser/parser"
	"strings"
)

func printTree(node *parser.Node, prefix string) {
	if node == nil {
		return
	}

	if node.Type == parser.TextNodeType && strings.TrimSpace(node.Data) == "" {
		return
	}

	info := ""
	switch node.Type {
	case parser.ElementNodeType:
		info = fmt.Sprintf("Elemento: <%s>", node.TagName)
		if len(node.Attributes) > 0 {
			attrs := []string{}
			for k, v := range node.Attributes {
				if v == "" {
					attrs = append(attrs, k)
				} else {
					attrs = append(attrs, fmt.Sprintf("%s=\"%s\"", k, v))
				}
			}
			info += fmt.Sprintf(" [Atributos: %s]", strings.Join(attrs, " "))
		}
	case parser.TextNodeType:
		info = fmt.Sprintf("Texto: \"%s\"", strings.TrimSpace(node.Data))
	case parser.CommentNodeType:
		info = fmt.Sprintf("Comentario: \"%s\"", strings.TrimSpace(node.Data))
	case parser.DoctypeNodeType:
		info = fmt.Sprintf("Doctype: %s", node.Data)
	case parser.DocumentFragmentNodeType:
		info = "Fragmento de Documento (Raíz)"
	default:
		info = "Nodo Desconocido"
	}
	fmt.Println(prefix + info)

	for _, child := range node.Children {
		printTree(child, prefix+"  ")
	}
}

func main() {
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Prueba de Resiliencia Total</title>
	</head>
	<body>
		<!-- Atributo mal formado y tag sin cerrar -->
		<div id="wrapper" / style="width:100%"> 

			<h1>Título Principal</b> <!-- Cierre incorrecto para h1 -->

			<p>Un párrafo con <em>énfasis incompleto.
			
			<br> <!-- Tag void sin auto-cierre, debe funcionar -->

		</div> <!-- Cierra el div, forzando cierres implícitos de h1 y p -->

		<p>Este párrafo está fuera de lugar.</p> <!-- No hay error aquí, es válido -->

		</i> <!-- Tag de cierre huérfano, sin apertura -->
	</body>
	</html>
	`

	fmt.Println("--- Parseando el siguiente HTML: ---")
	fmt.Println(html)

	doc, errs := parser.ParseHTML(html)
	fmt.Println("\n--- Estructura de Árbol Resultante: ---")
	if doc != nil {
		printTree(doc, "")
	} else {
		fmt.Println("El parseo resultó en un error fatal y no se pudo generar un árbol.")
	}

	if len(errs) > 0 {
		fmt.Println("\n--- Errores de Parseo Detectados: ---")
		for i, err := range errs {
			fmt.Printf("Error %d: %v\n", i+1, err)
		}
	} else {
		fmt.Println("\n--- Parseo completado sin errores. ---")
	}
}
