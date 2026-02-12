package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

type ClipData struct {
	Mime      string `form:"mime" json:"mime"`
	Timestamp int64  `form:"timestamp" json:"timestamp"`
	Content   string `form:"content" json:"content"`
}

var data ClipData

func main() {
	e := echo.New()

	e.GET("/clip", func(c echo.Context) error {
		return c.JSON(http.StatusOK, data)
	})

	e.POST("/clip", func(c echo.Context) error {
		c.Bind(&data)

		file, err := c.FormFile("file")
		if err == nil {
			f, err := file.Open()
			if err != nil {
				fmt.Println(err)
				return err
			}
			defer f.Close()

			bytes, err := io.ReadAll(f)
			if err != nil {
				fmt.Println(err)
				return err
			}

			data.Content = base64.StdEncoding.EncodeToString(bytes)
		}

		return c.String(http.StatusOK, "ok")
	})

	e.RouteNotFound("/*", func(c echo.Context) error {
		return c.NoContent(http.StatusForbidden)
	})

	e.Logger.Fatal(e.Start(":6233"))
}
