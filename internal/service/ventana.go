package service

import (
	"strings"

	"github.com/ArSteven/devconnect/internal/model"
)

const (
	lineasTarjeta  = 12 // líneas del código mejorado que muestra la tarjeta del feed
	margenOriginal = 48 // líneas extra del original cuando el cambio sigue más allá de la tarjeta
	lineasCambio   = 40 // tope del tramo del diff corto de la actividad (el navegador muestra 6)
)

// lineas parte un código como el diff del navegador: sin \r y sin saltos de línea al final.
func lineas(codigo string) []string {
	c := strings.TrimRight(strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(codigo), "\n")
	if c == "" {
		return []string{}
	}
	return strings.Split(c, "\n")
}

// prefijoComun cuenta las líneas iguales del comienzo: ahí empieza el primer cambio.
func prefijoComun(a, b []string) int {
	k := 0
	for k < len(a) && k < len(b) && a[k] == b[k] {
		k++
	}
	return k
}

// sufijoComun cuenta las líneas iguales del final.
func sufijoComun(a, b []string) int {
	k := 0
	for k < len(a) && k < len(b) && a[len(a)-1-k] == b[len(b)-1-k] {
		k++
	}
	return k
}

func unir(l []string, desde, hasta int) string {
	if desde >= hasta {
		return ""
	}
	return strings.Join(l[desde:hasta], "\n")
}

// ventanaTarjeta recorta el código mejorado a lo que muestra la tarjeta del feed: desde arriba
// si el cambio se ve, o desde dos líneas antes del primer cambio si quedaría más abajo.
// El original acompaña hasta un punto alineado (la parte final que no cambió), para que el
// diff del navegador marque exactamente las líneas aportadas.
func ventanaTarjeta(original, codigo string) model.Ventana {
	a, b := lineas(original), lineas(codigo)
	k := prefijoComun(a, b)
	s := sufijoComun(a[k:], b[k:])

	desde := 0
	if k > lineasTarjeta-4 {
		desde = max(0, min(k-2, len(b)-lineasTarjeta))
	}
	finB := min(len(b), desde+lineasTarjeta)
	var finA int
	if finB >= len(b)-s {
		finA = finB + len(a) - len(b) // la tarjeta llega a la parte final común: corte alineado
	} else {
		finA = min(len(a)-s, desde+lineasTarjeta+margenOriginal) // todo lo que cambió del original
	}
	return model.Ventana{Original: unir(a, desde, finA), Codigo: unir(b, desde, finB), DesdeLinea: desde + 1, TotalLineas: len(b)}
}

// ventanaCambio recorta los dos códigos a la zona que cambió, con una línea de contexto a cada
// lado: la actividad muestra ese diff corto sin traer los archivos completos.
func ventanaCambio(original, codigo string) model.Ventana {
	a, b := lineas(original), lineas(codigo)
	k := prefijoComun(a, b)
	s := sufijoComun(a[k:], b[k:])

	desde := max(0, k-1)
	finA := min(len(a)-s+1, len(a), desde+lineasCambio)
	finB := min(len(b)-s+1, len(b), desde+lineasCambio)
	return model.Ventana{Original: unir(a, desde, finA), Codigo: unir(b, desde, finB), DesdeLinea: desde + 1, TotalLineas: len(b)}
}
