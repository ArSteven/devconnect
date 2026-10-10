package service

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/ArSteven/devconnect/internal/model"
	"github.com/ArSteven/devconnect/internal/repository"
)

var ErrRangoInvalido = errors.New("la fecha desde no puede ser posterior a hasta")

// zonaColombia: Colombia no tiene horario de verano, así que un desfase fijo basta.
var zonaColombia = time.FixedZone("COT", -5*60*60)

type MetricasService struct {
	repo *repository.MetricaRepo
}

func NuevoMetricasService(r *repository.MetricaRepo) *MetricasService {
	return &MetricasService{repo: r}
}

// Calcular toma los días desde y hasta (AAAA-MM-DD, los dos incluidos) en hora de Colombia.
// Sin desde, cuenta toda la historia; sin hasta, llega hasta hoy.
func (s *MetricasService) Calcular(ctx context.Context, f model.FiltroMetricas) (*model.Metricas, error) {
	ahora := time.Now().In(zonaColombia)
	hasta := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, zonaColombia)
	if f.Hasta != "" {
		h, err := time.ParseInLocation(time.DateOnly, f.Hasta, zonaColombia)
		if err != nil {
			return nil, ErrRangoInvalido
		}
		hasta = h
	}
	var desde time.Time // el valor cero no pone límite inferior
	if f.Desde != "" {
		d, err := time.ParseInLocation(time.DateOnly, f.Desde, zonaColombia)
		if err != nil || d.After(hasta) {
			return nil, ErrRangoInvalido
		}
		desde = d
	}

	m, err := s.repo.Calcular(ctx, desde, hasta.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	m.Hasta = hasta.Format(time.DateOnly)
	if f.Desde != "" {
		d := desde.Format(time.DateOnly)
		m.Desde = &d
	}
	m.Retencion.Dia1.Tasa = porcentaje(m.Retencion.Dia1.Volvieron, m.Retencion.Dia1.Elegibles)
	m.Retencion.Dia7.Tasa = porcentaje(m.Retencion.Dia7.Volvieron, m.Retencion.Dia7.Elegibles)
	m.Empresas.TasaConversion = porcentaje(m.Empresas.Suscritas, m.Empresas.Registradas)
	return m, nil
}

// porcentaje con un decimal; 0 si no hay base.
func porcentaje(parte, total int) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(parte)*1000/float64(total)) / 10
}
