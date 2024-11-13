package main

import (
	"fmt"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"io"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strconv"
)

const name = "go.opentelemetry.io/otel/example/dice"

var (
	tracer  = otel.Tracer(name)
	meter   = otel.Meter(name)
	logger  = otelslog.NewLogger(name)
	rollCnt metric.Int64Counter
)

func init() {
	var err error
	rollCnt, err = meter.Int64Counter("dice.rolls",
		metric.WithDescription("number of dice rolls"),
		metric.WithUnit("{roll}"),
	)
	if err != nil {
		panic(err)
	}
}

func rolldice(w http.ResponseWriter, r *http.Request) {
	slog.SetDefault(logger)
	slog.
		logger.With(slog.NewTextHandler(os.Stdout, nil))
	ctx, span := tracer.Start(r.Context(), "roll")
	defer span.End()

	roll := 1 + rand.Intn(6)
	var msg string
	if player := r.PathValue("player"); player != "" {
		msg = fmt.Sprintf("%s is rolling the dice", player)
	} else {
		msg = "Anonymous player is rolling the dice"
	}
	logger.InfoContext(ctx, msg, "result", roll)
	logger.Error("errrrrrror", "ehsan", "2")

	rollValueAttr := attribute.Int("roll.value", roll)
	span.SetAttributes(rollValueAttr)
	rollCnt.Add(ctx, 1, metric.WithAttributes(rollValueAttr))

	resp := strconv.Itoa(roll) + "\n"
	if _, err := io.WriteString(w, resp); err != nil {
		log.Printf("Write failed: %v\n", err)
	}

}