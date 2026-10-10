package model

// EjecutarInput es lo que recibe POST /ejecutar/go. El tamaño máximo (64 KB) lo revisa el service.
type EjecutarInput struct {
	Codigo string `json:"codigo" binding:"required"`
}

// Salida es un tramo de lo que imprimió el programa. Flujo: stdout, stderr o aviso (lo que
// señala go vet: no impide que el programa corra).
type Salida struct {
	Flujo string `json:"flujo"`
	Texto string `json:"texto"`
}

// ResultadoEjecucion es la consola de «Pruébalo» para una ejecución.
type ResultadoEjecucion struct {
	Salida    []Salida `json:"salida"`
	Error     bool     `json:"error"`     // no compiló o terminó con un error
	Recortada bool     `json:"recortada"` // la salida pasaba de 10.000 caracteres
	Ms        int64    `json:"ms"`        // lo que tardó, ida y vuelta al Go Playground
}
