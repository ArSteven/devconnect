package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ArSteven/devconnect/internal/model"
)

const (
	// URLGoPlayground es la ejecución del Go Playground oficial (protocolo versión 2).
	URLGoPlayground = "https://go.dev/_/compile"
	MaxCodigoGo     = 64 << 10 // 64 KB
	MaxSalida       = 10_000   // caracteres que se muestran en la consola
	esperaGo        = 10 * time.Second
)

var (
	ErrCodigoGrande  = errors.New("el código pasa de 64 KB")
	ErrPlayground    = errors.New("el Go Playground no está disponible en este momento, intenta de nuevo")
	ErrTiempoAgotado = errors.New("el Go Playground no respondió en 10 segundos, intenta de nuevo")
)

// EjecucionService corre código Go en el Go Playground oficial. DevConnect nunca compila ni ejecuta
// código en su servidor: solo reenvía el texto y devuelve lo que el Playground responde.
type EjecucionService struct {
	cliente *http.Client
	url     string
	espera  time.Duration
}

func NuevoEjecucionService(urlPlayground string) *EjecucionService {
	return &EjecucionService{cliente: &http.Client{}, url: urlPlayground, espera: esperaGo}
}

// respuestaPlayground es lo que devuelve /_/compile (llega como text/plain, pero es JSON).
type respuestaPlayground struct {
	Errors    string // errores de compilación
	VetErrors string // avisos de go vet: el programa igual corre
	Events    []struct {
		Message string
		Kind    string // stdout o stderr
	}
}

func (s *EjecucionService) Go(ctx context.Context, codigo string) (*model.ResultadoEjecucion, error) {
	if len(codigo) > MaxCodigoGo {
		return nil, ErrCodigoGrande
	}
	ctx, cancelar := context.WithTimeout(ctx, s.espera)
	defer cancelar()

	form := url.Values{"version": {"2"}, "body": {codigo}, "withVet": {"true"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "DevConnect (+https://github.com/ArSteven/devconnect)")

	inicio := time.Now()
	resp, err := s.cliente.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, ErrTiempoAgotado
		}
		return nil, fmt.Errorf("%w: %v", ErrPlayground, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: respondió %d", ErrPlayground, resp.StatusCode)
	}
	var r respuestaPlayground
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&r); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, ErrTiempoAgotado
		}
		return nil, fmt.Errorf("%w: %v", ErrPlayground, err)
	}
	res := traducirPlayground(r)
	res.Ms = time.Since(inicio).Milliseconds()
	return res, nil
}

// traducirPlayground arma la consola: errores de compilación o lo que imprimió el programa, en orden.
// Un programa con salida en stderr (un panic, log.Fatal) cuenta como terminado con error.
func traducirPlayground(r respuestaPlayground) *model.ResultadoEjecucion {
	res := &model.ResultadoEjecucion{Salida: []model.Salida{}}
	agregar := func(flujo, texto string) {
		if texto == "" {
			return
		}
		if n := len(res.Salida); n > 0 && res.Salida[n-1].Flujo == flujo {
			res.Salida[n-1].Texto += texto
			return
		}
		res.Salida = append(res.Salida, model.Salida{Flujo: flujo, Texto: texto})
	}
	if r.Errors != "" {
		res.Error = true
		agregar("stderr", r.Errors)
	}
	for _, e := range r.Events {
		flujo := "stdout"
		if e.Kind == "stderr" {
			flujo = "stderr"
			res.Error = true
		}
		agregar(flujo, e.Message)
	}
	if r.VetErrors != "" {
		agregar("aviso", "go vet: "+r.VetErrors)
	}
	res.Salida, res.Recortada = recortarSalida(res.Salida, MaxSalida)
	return res
}

// recortarSalida deja como mucho max caracteres en total y avisa si cortó algo.
func recortarSalida(salida []model.Salida, max int) ([]model.Salida, bool) {
	quedan := max
	for i := range salida {
		texto := []rune(salida[i].Texto)
		if len(texto) <= quedan {
			quedan -= len(texto)
			continue
		}
		salida[i].Texto = string(texto[:quedan])
		if quedan == 0 {
			return salida[:i], true
		}
		return salida[:i+1], true
	}
	return salida, false
}
