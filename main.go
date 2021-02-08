package main

import (
	"bytes"
	"github.com/labstack/echo"
	"github.com/tidwall/gjson"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

func convert(code string, hex bool, thumb bool) string {

	var arch, mode_before, mode_after string

	url := "https://armconverter.com/api/convert"

	if hex {
		mode_before = "hex"
		mode_after = "asm"
	} else {
		mode_before = "asm"
		mode_after = "hex"
	}

	if thumb {
		arch = "thumbbe"
	} else {
		arch = "armbe"
	}

	body := `{"` + mode_before + `":"` + code + `","offset":"","arch":"` + arch + `"}`

	header := map[string]string{
		"Host":            "armconverter.com",
		"Content-Length":  strconv.Itoa(len(body)),
		"Content-Type":    "application/json",
		"Accept":          "*/*",
		"Accept-Encofing": "gzip, deflate, br",
		"Connection":      "keep-alive",
	}

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))

	for key, value := range header {
		req.Header.Set(key, value)
	}

	client := new(http.Client)
	resp, _ := client.Do(req)

	defer resp.Body.Close()

	json, _ := ioutil.ReadAll(resp.Body)

	result, _ := strconv.ParseBool(gjson.Get(string(json), mode_after+"."+arch+".0").String())

	if !result {
		return "Could not disassemble"
	}

	return gjson.Get(string(json), mode_after+"."+arch+".1").String()
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {

	t := &Template{
		templates: template.Must(template.ParseGlob("*.html")),
	}

	e := echo.New()
	e.Static("/", "")
	e.Renderer = t
	e.GET("/", index)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	e.Start(":" + port)
}

func index(c echo.Context) error {

	var mode, arm, thumb string

	code := c.QueryParam("code")

	if code == "nop" {
		mode = "ASM"
		arm = convert("mov r0, r0", false, false)
		thumb = convert("mov r8, r8", false, true)
	} else if len(code) >= 4 {
		_, err := strconv.ParseInt(code[:4], 16, 32)

		if err == nil {
			mode = "HEX"
			arm = convert(code, true, false)
			thumb = convert(code, true, true)
		} else {
			mode = "ASM"
			arm = convert(code, false, false)
			thumb = convert(code, false, true)
		}
	} else if len(code) != 0 {
		arm = "Could not disassemble"
		thumb = "Could not disassemble"
	}

	data := map[string]interface{}{
		"type":  mode,
		"code":  code,
		"arm":   arm,
		"thumb": thumb,
	}

	return c.Render(http.StatusOK, "index.html", data)
}
