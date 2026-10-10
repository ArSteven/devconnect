package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ArSteven/devconnect/internal/model"
)

func TestTraducirPlayground(t *testing.T) {
	casos := []struct {
		nombre    string
		json      string
		salida    []model.Salida
		conError  bool
		recortada bool
	}{
		{
			nombre: "programa que imprime",
			json:   `{"Errors":"","Events":[{"Message":"hola\n","Kind":"stdout","Delay":0},{"Message":"mundo\n","Kind":"stdout","Delay":0}],"VetErrors":""}`,
			salida: []model.Salida{{Flujo: "stdout", Texto: "hola\nmundo\n"}},
		},
		{
			nombre:   "error de compilación",
			json:     `{"Errors":"./prog.go:4:2: declared and not used: x\n","Events":null,"VetErrors":""}`,
			salida:   []model.Salida{{Flujo: "stderr", Texto: "./prog.go:4:2: declared and not used: x\n"}},
			conError: true,
		},
		{
			nombre:   "panic después de imprimir",
			json:     `{"Errors":"","Events":[{"Message":"antes\n","Kind":"stdout"},{"Message":"panic: assignment to entry in nil map\n","Kind":"stderr"}],"VetErrors":""}`,
			salida:   []model.Salida{{Flujo: "stdout", Texto: "antes\n"}, {Flujo: "stderr", Texto: "panic: assignment to entry in nil map\n"}},
			conError: true,
		},
		{
			nombre: "aviso de go vet: el programa corre y no cuenta como error",
			json:   `{"Errors":"","Events":[{"Message":"%!d(string=texto)\n","Kind":"stdout"}],"VetErrors":"prog.go:6:14: fmt.Printf format %d has arg \"texto\" of wrong type string\n"}`,
			salida: []model.Salida{
				{Flujo: "stdout", Texto: "%!d(string=texto)\n"},
				{Flujo: "aviso", Texto: "go vet: prog.go:6:14: fmt.Printf format %d has arg \"texto\" of wrong type string\n"},
			},
		},
		{
			nombre:    "salida de más de 10.000 caracteres",
			json:      `{"Events":[{"Message":"` + strings.Repeat("a", 9_990) + `","Kind":"stdout"},{"Message":"` + strings.Repeat("b", 50) + `","Kind":"stderr"}]}`,
			salida:    []model.Salida{{Flujo: "stdout", Texto: strings.Repeat("a", 9_990)}, {Flujo: "stderr", Texto: strings.Repeat("b", 10)}},
			conError:  true,
			recortada: true,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				io.WriteString(w, c.json)
			}))
			defer srv.Close()
			res, err := NuevoEjecucionService(srv.URL).Go(context.Background(), "package main")
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if res.Error != c.conError || res.Recortada != c.recortada {
				t.Errorf("error=%v recortada=%v, se esperaba error=%v recortada=%v", res.Error, res.Recortada, c.conError, c.recortada)
			}
			if len(res.Salida) != len(c.salida) {
				t.Fatalf("salida = %#v, se esperaba %#v", res.Salida, c.salida)
			}
			for i := range c.salida {
				if res.Salida[i] != c.salida[i] {
					t.Errorf("salida[%d] = %#v, se esperaba %#v", i, res.Salida[i], c.salida[i])
				}
			}
		})
	}
}

func TestGoEnviaElCodigoAlPlayground(t *testing.T) {
	var recibido url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("petición inesperada: %s %s", r.Method, r.Header.Get("Content-Type"))
		}
		r.ParseForm()
		recibido = r.PostForm
		io.WriteString(w, `{"Errors":"","Events":[]}`)
	}))
	defer srv.Close()
	codigo := "package main\n\nfunc main() {}\n"
	if _, err := NuevoEjecucionService(srv.URL).Go(context.Background(), codigo); err != nil {
		t.Fatal(err)
	}
	if recibido.Get("version") != "2" || recibido.Get("body") != codigo || recibido.Get("withVet") != "true" {
		t.Errorf("el Playground recibió %v", recibido)
	}
}

func TestGoFallas(t *testing.T) {
	lento := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(500 * time.Millisecond):
		case <-r.Context().Done():
		}
	}))
	defer lento.Close()
	caido := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fuera de servicio", http.StatusServiceUnavailable)
	}))
	defer caido.Close()
	raro := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html>no es JSON</html>")
	}))
	defer raro.Close()

	s := NuevoEjecucionService(lento.URL)
	s.espera = 100 * time.Millisecond
	if _, err := s.Go(context.Background(), "package main"); !errors.Is(err, ErrTiempoAgotado) {
		t.Errorf("Playground lento: %v, se esperaba ErrTiempoAgotado", err)
	}
	if _, err := NuevoEjecucionService(caido.URL).Go(context.Background(), "package main"); !errors.Is(err, ErrPlayground) {
		t.Errorf("Playground caído: %v, se esperaba ErrPlayground", err)
	}
	if _, err := NuevoEjecucionService(raro.URL).Go(context.Background(), "package main"); !errors.Is(err, ErrPlayground) {
		t.Errorf("respuesta que no es JSON: %v, se esperaba ErrPlayground", err)
	}
	if _, err := NuevoEjecucionService(raro.URL).Go(context.Background(), strings.Repeat("x", MaxCodigoGo+1)); !errors.Is(err, ErrCodigoGrande) {
		t.Errorf("código de más de 64 KB: %v, se esperaba ErrCodigoGrande", err)
	}
}
