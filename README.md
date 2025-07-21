# HTML Parser en Go

Este proyecto es una implementación de un parser de HTML simplificado escrito completamente en Go, sin utilizar bibliotecas externas como `golang.org/x/net/html`. El objetivo es tomar una cadena de HTML y transformarla en una estructura de árbol de nodos jerárquica (DOM).

El parser está diseñado con un enfoque en la **resiliencia**, lo que le permite construir un árbol de "mejor esfuerzo" a partir de HTML imperfecto, mientras **recolecta y reporta los errores de sintaxis** encontrados.

## Estructura del Proyecto

```
html-parser/
├── go.mod                # Define el módulo del proyecto
├── main.go               # Programa ejecutable para visualizar el árbol parseado
└── parser/               # El paquete de la librería que contiene la lógica del parser
    ├── parser.go         # La implementación principal del parser y las estructuras de nodos
    └── parser_test.go    # Las pruebas unitarias para el paquete parser
```

## Enfoque General del Parseo

El parser utiliza un enfoque de **un solo paso (single-pass)** con una **máquina de estados implícita**. En lugar de tener una fase de "lexing" (tokenización) y otra de "parsing" separadas, el parser lee la cadena de entrada carácter por carácter y toma decisiones basadas en el contexto actual (si está dentro de una etiqueta, leyendo texto, etc.).

El estado se gestiona en una `struct parser` interna que rastrea:
- La posición de lectura actual (`pos`).
- La cadena de entrada (`input`).
- Una pila de nodos para el anidamiento (`stack`).
- Un slice para recolectar errores no fatales (`errors`).

## Construcción del Árbol de Nodos y Manejo de Anidación

La clave para manejar la jerarquía y el anidamiento de etiquetas es una **pila de nodos** (`[]*Node`).

1.  **Inicio:** Se inicia con un nodo raíz virtual (`DocumentFragmentNodeType`) en la pila.
2.  **Etiqueta de Apertura (`<tag>`):** Se crea un nuevo `Node`. Este nodo se añade como hijo del elemento que está en la cima de la pila. A continuación, el nuevo nodo es empujado (push) a la cima de la pila, convirtiéndose en el nuevo padre para los nodos siguientes.
3.  **Etiqueta de Auto-cierre (`<tag/>`):** Se crea un nuevo `Node` y se añade como hijo del elemento en la cima de la pila, pero **no** se empuja a la pila. Esto asegura que los elementos siguientes sean sus hermanos, no sus hijos.
4.  **Etiqueta de Cierre (`</tag>`):** Se comprueba si el nombre de la etiqueta de cierre coincide con la etiqueta en la cima de la pila. Si coinciden, el nodo se saca (pop) de la pila, "subiendo" un nivel en el árbol. Si no coinciden, se registra un error de anidamiento y la etiqueta de cierre es ignorada para maximizar la resiliencia.

## Estrategia para Atributos y Texto

-   **Texto:** El texto se identifica como cualquier secuencia de caracteres que no esté dentro de una etiqueta (`<...>`). El parser consume estos caracteres y los agrupa en un `Node` de tipo `TextNodeType`.
-   **Atributos:** Cuando el parser está dentro de una etiqueta de apertura (después del nombre de la etiqueta pero antes del `>`), entra en un modo de análisis de atributos. Busca pares `clave=valor` y es lo suficientemente flexible para manejar:
    -   Valores con comillas dobles (`class="main"`).
    -   Valores con comillas simples (`class='main'`).
    -   Valores sin comillas (`id=main`).
    -   Atributos booleanos/sin valor (`disabled`).

## Resiliencia a Errores

El parser distingue entre **errores fatales** (que detienen el parseo) y **errores recuperables** (que son registrados).

#### Errores Fatales
Estos errores detienen el análisis porque el parser pierde el contexto estructural. `ParseHTML` devolverá un `*Node` nulo.
-   `ErrUnterminatedTag`: Una etiqueta o un valor de atributo entre comillas nunca se cierra.
-   `ErrUnterminatedComment`: Un comentario `<!--` nunca se cierra con `-->`.
-   `ErrEmptyInput`: La cadena de entrada está vacía.

#### Errores Recuperables (No Fatales)
El parser registra estos errores y continúa analizando el resto del documento para construir el mejor árbol posible.
-   **Atributos mal formados:** Caracteres inesperados entre atributos (ej. `<div id="a" / class="b">`). El carácter inválido es registrado como un error y el parser continúa buscando el siguiente atributo.
-   **Etiquetas de cierre incorrectas:** Una etiqueta de cierre que no coincide con la última etiqueta abierta (ej. `<b><i>texto</b>`). El error se registra y la etiqueta de cierre se ignora.