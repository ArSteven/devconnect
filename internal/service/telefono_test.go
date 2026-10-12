package service

import "testing"

func TestNormalizarTelefono(t *testing.T) {
	casos := []struct {
		nombre, entrada, esperado string
		valido                    bool
	}{
		{"celular con espacios y guion", "310 482-7156", "3104827156", true},
		{"con indicativo y paréntesis", "+57 (310) 482 7156", "+573104827156", true},
		{"fijo de siete dígitos", "6345678", "6345678", true},
		{"vacío: borra el teléfono", "   ", "", true},
		{"muy corto", "12345", "", false},
		{"muy largo", "1234567890123456", "", false},
		{"con letras", "310abc7156", "", false},
		{"más en medio", "310+4827156", "", false},
	}
	for _, c := range casos {
		obtenido, valido := normalizarTelefono(c.entrada)
		if obtenido != c.esperado || valido != c.valido {
			t.Errorf("%s: normalizarTelefono(%q) = %q, %v; se esperaba %q, %v",
				c.nombre, c.entrada, obtenido, valido, c.esperado, c.valido)
		}
	}
}
