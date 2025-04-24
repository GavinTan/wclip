package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type ClipData struct {
	// windows: 1  darwin: 2  linux: 3  android: 4  ios: 5
	Type    int    `json:"type"`
	Content string `json:"content"`
}

var data ClipData

func main() {
	e := echo.New()

	e.GET("/clip", func(c echo.Context) error {
		return c.JSON(http.StatusOK, data)
	})

	e.POST("/clip", func(c echo.Context) error {
		c.Bind(&data)
		return c.String(http.StatusOK, "ok")
	})

	e.RouteNotFound("/*", func(c echo.Context) error {
		return c.NoContent(http.StatusForbidden)
	})

	e.Logger.Fatal(e.Start(":6233"))
}
