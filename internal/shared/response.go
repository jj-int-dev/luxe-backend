package shared

import "github.com/labstack/echo/v4"

func JSON(c echo.Context, status int, data interface{}) error {
    return c.JSON(status, data)
}