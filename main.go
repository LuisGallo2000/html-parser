package main

import (
	"fmt"
	"html-parser/parser"
	"log"
	"os"
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
	filename := "prueba.html"

	content, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error al leer el archivo '%s': %v", filename, err)
	}

	htmlContent := string(content)

	fmt.Println("--- Parseando el contenido del HTML: ---")
	fmt.Println(htmlContent)

	doc, errs := parser.ParseHTML(htmlContent)

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
