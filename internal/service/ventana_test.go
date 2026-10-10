package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ArSteven/devconnect/internal/model"
)

// numeradas devuelve "L1".."Ln" como líneas de código.
func numeradas(n int) []string {
	l := make([]string, n)
	for i := range l {
		l[i] = fmt.Sprintf("L%d", i+1)
	}
	return l
}

func codigo(l ...string) string { return strings.Join(l, "\n") }

func TestVentanaTarjeta(t *testing.T) {
	veinte := numeradas(20)
	cuarenta := numeradas(40)
	conCambio := append(append(append([]string{}, cuarenta[:29]...), "Y"), cuarenta[30:]...)

	casos := []struct {
		nombre           string
		original, mejora string
		esperada         model.Ventana
	}{
		{
			nombre:   "inserción al comienzo: se ve desde arriba y el original corta alineado",
			original: codigo(veinte...),
			mejora:   codigo(append([]string{"L1", "L2", "X"}, veinte[2:]...)...),
			esperada: model.Ventana{Original: codigo(veinte[:11]...), Codigo: codigo(append([]string{"L1", "L2", "X"}, veinte[2:11]...)...), DesdeLinea: 1, TotalLineas: 21},
		},
		{
			nombre:   "cambio en la línea 30: la tarjeta empieza dos líneas antes",
			original: codigo(cuarenta...),
			mejora:   codigo(conCambio...),
			esperada: model.Ventana{Original: codigo(cuarenta[27:39]...), Codigo: codigo(conCambio[27:39]...), DesdeLinea: 28, TotalLineas: 40},
		},
		{
			nombre:   "reto sin código inicial: todo es aporte",
			original: "",
			mejora:   codigo(numeradas(15)...),
			esperada: model.Ventana{Original: "", Codigo: codigo(numeradas(12)...), DesdeLinea: 1, TotalLineas: 15},
		},
		{
			nombre:   "borrado al comienzo: el original trae lo borrado para alinear",
			original: codigo(append([]string{"D1", "D2", "D3", "D4", "D5"}, veinte...)...),
			mejora:   codigo(veinte...),
			esperada: model.Ventana{Original: codigo(append([]string{"D1", "D2", "D3", "D4", "D5"}, veinte[:12]...)...), Codigo: codigo(veinte[:12]...), DesdeLinea: 1, TotalLineas: 20},
		},
		{
			nombre:   "saltos de línea de Windows y al final no cuentan como cambio",
			original: "a\r\nb\r\n",
			mejora:   "a\nb",
			esperada: model.Ventana{Original: "a\nb", Codigo: "a\nb", DesdeLinea: 1, TotalLineas: 2},
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if v := ventanaTarjeta(c.original, c.mejora); v != c.esperada {
				t.Errorf("\nobtuvo   %+v\nesperaba %+v", v, c.esperada)
			}
		})
	}
}

func TestVentanaCambio(t *testing.T) {
	cuarenta := numeradas(40)
	conCambio := append(append(append([]string{}, cuarenta[:29]...), "Y"), cuarenta[30:]...)

	casos := []struct {
		nombre           string
		original, mejora string
		esperada         model.Ventana
	}{
		{
			nombre:   "un reemplazo: la línea de antes, el cambio y la de después",
			original: codigo(cuarenta...),
			mejora:   codigo(conCambio...),
			esperada: model.Ventana{Original: "L29\nL30\nL31", Codigo: "L29\nY\nL31", DesdeLinea: 29, TotalLineas: 40},
		},
		{
			nombre:   "cambio en la primera línea: sin contexto antes",
			original: codigo("a", "b", "c"),
			mejora:   codigo("x", "b", "c"),
			esperada: model.Ventana{Original: "a\nb", Codigo: "x\nb", DesdeLinea: 1, TotalLineas: 3},
		},
		{
			nombre:   "sin cambios: queda la última línea y el navegador no muestra diff",
			original: codigo("a", "b"),
			mejora:   codigo("a", "b"),
			esperada: model.Ventana{Original: "b", Codigo: "b", DesdeLinea: 2, TotalLineas: 2},
		},
		{
			nombre:   "un cambio enorme se corta en el tope",
			original: "",
			mejora:   codigo(numeradas(100)...),
			esperada: model.Ventana{Original: "", Codigo: codigo(numeradas(lineasCambio)...), DesdeLinea: 1, TotalLineas: 100},
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if v := ventanaCambio(c.original, c.mejora); v != c.esperada {
				t.Errorf("\nobtuvo   %+v\nesperaba %+v", v, c.esperada)
			}
		})
	}
}
